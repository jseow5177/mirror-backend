package handler

import (
	"bufio"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"mime/multipart"
	"time"
)

type TaskHandler interface {
	CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error
	GetFileUploadTasks(ctx context.Context, req *GetFileUploadTasksRequest, res *GetFileUploadTasksResponse) error
}

type taskHandler struct {
	taskRepo repo.TaskRepo
	fileRepo repo.FileRepo
}

func NewTaskHandler(taskRepo repo.TaskRepo, fileRepo repo.FileRepo) TaskHandler {
	return &taskHandler{
		taskRepo,
		fileRepo,
	}
}

type GetFileUploadTasksRequest struct {
	ContextInfo

	ResourceID   *uint64          `json:"resource_id,omitempty"`
	ResourceType *uint32          `json:"resource_type,omitempty"`
	Pagination   *repo.Pagination `json:"pagination,omitempty"`
}

func (req *GetFileUploadTasksRequest) GetResourceID() uint64 {
	if req != nil && req.ResourceID != nil {
		return *req.ResourceID
	}
	return 0
}

func (req *GetFileUploadTasksRequest) GetResourceType() uint32 {
	if req != nil && req.ResourceType != nil {
		return *req.ResourceType
	}
	return 0
}

type GetFileUploadTasksResponse struct {
	Tasks      []*entity.Task   `json:"tasks"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

var GetFileUploadTasksValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
	"resource_id": &validator.UInt64{
		Optional: false,
	},
	"resource_type": &validator.UInt32{
		Optional: false,
	},
	"pagination": PaginationValidator(),
})

func (h *taskHandler) GetFileUploadTasks(ctx context.Context, req *GetFileUploadTasksRequest, res *GetFileUploadTasksResponse) error {
	if err := GetFileUploadTasksValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.Pagination == nil {
		req.Pagination = new(repo.Pagination)
	}

	tasks, pagination, err := h.taskRepo.GetByResourceIDAndType(ctx, req.GetResourceID(), entity.ResourceType(req.GetResourceType()), req.Pagination)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tasks failed: %v", err)
		return err
	}

	res.Tasks = tasks
	res.Pagination = pagination

	return nil
}

type CreateFileUploadTaskRequest struct {
	ContextInfo
	FileUpload

	ResourceID   *uint64 `schema:"resource_id,required"`
	ResourceType *uint32 `schema:"resource_type,required"`
}

func (req *CreateFileUploadTaskRequest) GetResourceType() uint32 {
	if req != nil && req.ResourceType != nil {
		return *req.ResourceType
	}
	return 0
}

func (req *CreateFileUploadTaskRequest) ToTask(extInfo *entity.TaskExtInfo) *entity.Task {
	if extInfo == nil {
		extInfo = new(entity.TaskExtInfo)
	}

	now := time.Now()
	return &entity.Task{
		ResourceID:   req.ResourceID,
		ResourceType: entity.ResourceType(req.GetResourceType()),
		Status:       entity.TaskStatusPending,
		TaskType:     entity.TaskTypeFileUpload,
		ExtInfo:      extInfo,
		CreatorID:    goutil.Uint64(req.GetUserID()),
		CreateTime:   goutil.Uint64(uint64(now.Unix())),
		UpdateTime:   goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateFileUploadTaskResponse struct {
	Task *entity.Task `json:"task,omitempty"`
}

var CreateFileUploadTaskValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator,
	"FileUpload": FileUploadValidator(false, 5_000_000, []string{
		"text/csv",
		"text/plain",
	}),
	"resource_id": &validator.UInt64{
		Optional: false,
	},
	"resource_type": &validator.UInt32{
		Optional:   false,
		Validators: []validator.UInt32Func{entity.CheckResourceType},
	},
})

func (h *taskHandler) CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error {
	if err := CreateFileUploadTaskValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	suffix, err := goutil.GenerateRandomString(16)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("generate random string failed: %v", err)
		return err
	}

	fileName := fmt.Sprintf("%s-%s-%d",
		req.GetFileName(),
		goutil.Base64Encode(suffix),
		time.Now().Unix(),
	)

	size, err := h.countRows(req.GetFile())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("count rows failed: %v", err)
		return err
	}

	fileID, err := h.fileRepo.CreateFile(ctx, goutil.String(req.GetTenantFolder()), fileName, req.GetFile())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create file failed: %v", err)
		return err
	}

	task := req.ToTask(&entity.TaskExtInfo{
		FileID:      goutil.String(fileID),
		OriFileName: goutil.String(req.GetFileName()),
		Progress:    goutil.Uint64(0),
		Size:        goutil.Uint64(size - 1), // exclude header
	})
	id, err := h.taskRepo.Create(ctx, task)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create file upload task failed: %v", err)
		return err
	}

	task.ID = goutil.Uint64(id)
	res.Task = task

	return nil
}

func (h *taskHandler) countRows(file multipart.File) (uint64, error) {
	var (
		scanner = bufio.NewScanner(file)
		count   = uint64(0)
	)

	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return count, nil
}
