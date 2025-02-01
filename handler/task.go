package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"github.com/rs/zerolog/log"
	"time"
)

type TaskHandler interface {
	CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error
}

type taskHandler struct {
	taskRepo repo.TaskRepo
}

func NewTaskHandler(taskRepo repo.TaskRepo) TaskHandler {
	return &taskHandler{
		taskRepo,
	}
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

func (req *CreateFileUploadTaskRequest) ToTask() *entity.Task {
	now := time.Now()
	return &entity.Task{
		ResourceID:   req.ResourceID,
		ResourceType: entity.ResourceType(req.GetResourceType()),
		Status:       entity.TaskStatusPending,
		TaskType:     entity.TaskTypeFileUpload,
		ExtInfo:      &entity.TaskExtInfo{},
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

	task := req.ToTask()
	id, err := h.taskRepo.Create(ctx, task)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create file upload task failed: %v", err)
		return err
	}

	task.ID = goutil.Uint64(id)
	res.Task = task

	return nil
}
