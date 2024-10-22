package handler

import (
	"bufio"
	"bytes"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/mq"
	"cdp/pkg/router"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"time"
)

type TaskHandler interface {
	CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error
	WriteFileToStorage(ctx context.Context, req *mq.NotifyCreateTask) error
}

type taskHandler struct {
	fileRepo         repo.FileRepo
	taskRepo         repo.TaskRepo
	profileRepo      repo.ProfileRepo
	queryRepo        repo.QueryRepo
	mappingIDHandler MappingIDHandler
	producer         *mq.Producer
}

func NewTaskHandler(fileRepo repo.FileRepo, taskRepo repo.TaskRepo,
	profileRepo repo.ProfileRepo, queryRepo repo.QueryRepo, mappingIDHandler MappingIDHandler, producer *mq.Producer) TaskHandler {
	return &taskHandler{
		fileRepo:         fileRepo,
		taskRepo:         taskRepo,
		profileRepo:      profileRepo,
		queryRepo:        queryRepo,
		mappingIDHandler: mappingIDHandler,
		producer:         producer,
	}
}

type CreateFileUploadTaskRequest struct {
	*router.FileInfo `json:"file_info,omitempty"`

	ProfileID    *uint64 `schema:"profile_id" json:"profile_id,omitempty"`
	ProfileValue *string `schema:"profile_value" json:"profile_value,omitempty"`
	ProfileType  *uint32 `schema:"profile_type" json:"profile_type,omitempty"`
	Action       *uint32 `schema:"action" json:"action,omitempty"`
}

func (req *CreateFileUploadTaskRequest) GetAction() uint32 {
	if req != nil && req.Action != nil {
		return *req.Action
	}
	return 0
}

func (req *CreateFileUploadTaskRequest) GetProfileID() uint64 {
	if req != nil && req.ProfileID != nil {
		return *req.ProfileID
	}
	return 0
}

func (req *CreateFileUploadTaskRequest) GetProfileType() uint32 {
	if req != nil && req.ProfileType != nil {
		return *req.ProfileType
	}
	return 0
}

type CreateFileUploadTaskResponse struct {
	Task *entity.Task `json:"task,omitempty"`
}

var CreateFileUploadTaskValidator = validator.MustForm(map[string]validator.Validator{
	"profile_id": &validator.UInt64{},
	"profile_value": &validator.String{
		Optional: true,
	},
	"profile_type": &validator.UInt32{
		Min: goutil.Uint32(uint32(entity.ProfileTypeTag)),
		Max: goutil.Uint32(uint32(entity.ProfileTypeTag)),
	},
	"action": &validator.UInt32{
		Optional: true,
		Min:      goutil.Uint32(uint32(entity.TaskActionAdd)),
		Max:      goutil.Uint32(uint32(entity.TaskActionDelete)),
	},
	"file_info": FileInfoValidator(false, 5<<24, []string{"text/plain", "text/csv"}),
})

func (h *taskHandler) CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error {
	err := CreateFileUploadTaskValidator.Validate(req)
	if err != nil {
		return errutil.ValidationError(err)
	}

	now := time.Now().Unix()
	task := &entity.Task{
		TagID:      nil, // to be set by PreCreate
		TagValue:   nil, // to be set by PreCreate
		FileName:   goutil.String(req.FileHeader.Filename),
		FileKey:    goutil.String(h.generateFileKey(req.FileHeader.Filename)),
		Status:     goutil.Uint32(uint32(entity.TaskStatusPending)),
		URL:        goutil.String(""),
		Action:     req.Action,
		CreateTime: goutil.Uint64(uint64(now)),
		UpdateTime: goutil.Uint64(uint64(now)),
	}

	c := NewTaskCreator(h.profileRepo, req.GetProfileID(), req.GetProfileType(), req.ProfileValue)
	task, err = c.PreCreate(ctx, task)
	if err != nil {
		return errutil.ValidationError(err)
	}

	url, err := h.fileRepo.Upload(ctx, task.GetFileKey(), req.File)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("upload file %s failed, err: %v", task.GetFileName(), err)
		return err
	}
	task.URL = goutil.String(url)

	id, err := h.taskRepo.Create(ctx, task)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create task failed: %v", err)
		return err
	}
	task.ID = goutil.Uint64(id)

	// no need to return error
	go func() {
		for i := 0; i < 5; i++ {
			if err := h.producer.SendMessage(&mq.Message{
				Payload: mq.PayloadNotifyCreateTask,
				Key:     fmt.Sprint(req.GetProfileID()),
				Body: &mq.NotifyCreateTask{
					TaskID: goutil.Uint64(id),
				},
			}); err != nil {
				log.Ctx(ctx).Error().Msgf("send message failed: %v, taskID: %v, try: %v", err, task.GetID(), i+1)
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		}
	}()

	res.Task = task

	return nil
}

func (h *taskHandler) WriteFileToStorage(ctx context.Context, req *mq.NotifyCreateTask) error {
	taskFilter := &repo.TaskFilter{
		ID: req.TaskID,
	}
	task, err := h.taskRepo.Get(ctx, taskFilter)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get task failed: %v", err)
		return err
	}

	if !task.IsPending() {
		return fmt.Errorf("task is not pending, taskID: %d", task.GetID())
	}

	tagRepo := h.profileRepo.ToTagRepo()
	tag, err := tagRepo.Get(ctx, &repo.TagFilter{
		ID: task.TagID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tag failed: %v", err)
		return err
	}

	defer func() {
		taskStatus := entity.TaskStatusSuccess
		if err != nil {
			taskStatus = entity.TaskStatusFailed
		}

		err = h.taskRepo.Update(ctx, taskFilter, &entity.Task{
			Status: goutil.Uint32(uint32(taskStatus)),
		})
		if err != nil {
			log.Ctx(ctx).Error().Msgf("update task status failed: %v, status: %v", err, taskStatus)
		}
	}()

	var b []byte
	b, err = h.fileRepo.Download(ctx, task.GetFileKey())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("download file %s failed: %v", task.GetFileKey(), err)
		return err
	}

	// set task to processing
	err = h.taskRepo.Update(ctx, taskFilter, &entity.Task{
		Status: goutil.Uint32(uint32(entity.TaskStatusProcessing)),
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("update task to processing failed: %v", err)
		return err
	}

	scanner := bufio.NewScanner(bytes.NewReader(b))
	udIDs := make([]string, 0)

	for scanner.Scan() {
		udIDs = append(udIDs, scanner.Text())
	}

	mappingIDs := make([]*entity.MappingID, 0, len(udIDs))
	for i := 0; i < len(udIDs); i += MappingIDsMaxSize {
		end := i + MappingIDsMaxSize
		if end > len(udIDs) {
			end = len(udIDs)
		}

		req := &GetSetMappingIDsRequest{
			UdIDs: udIDs[i:end],
		}
		res := new(GetSetMappingIDsResponse)

		if err := h.mappingIDHandler.GetSetMappingIDs(ctx, req, res); err != nil {
			log.Ctx(ctx).Error().Msgf("get set mapping ids failed: %v", err)
			return err
		}

		mappingIDs = append(mappingIDs, res.MappingIDs...)
	}

	udTags := make([]*entity.UdTag, 0, len(mappingIDs))
	for _, mappingID := range mappingIDs {
		udTags = append(udTags, &entity.UdTag{
			MappingID: mappingID,
			Tag:       tag,
			TagValue:  task.TagValue,
		})
	}

	if err := h.queryRepo.Insert(ctx, udTags); err != nil {
		log.Ctx(ctx).Error().Msgf("insert ud tags failed: %v", err)
		return err
	}

	return nil
}

func (h *taskHandler) generateFileKey(fileName string) string {
	hashKey := fmt.Sprintf("%s-%d", fileName, time.Now().Unix())

	hFn := md5.New()
	hFn.Write([]byte(hashKey))
	hashValue := hFn.Sum(nil)

	return fmt.Sprintf("f-%s", hex.EncodeToString(hashValue))
}

type TaskCreator interface {
	PreCreate(ctx context.Context, task *entity.Task) (*entity.Task, error)
}

func NewTaskCreator(profileRepo repo.ProfileRepo, profileID uint64, profileType uint32, profileValue *string) TaskCreator {
	switch profileType {
	case uint32(entity.ProfileTypeTag):
		return &tagTaskCreator{tagRepo: profileRepo.ToTagRepo(), profileID: profileID, profileValue: profileValue}
	}
	panic(fmt.Sprintf("task creator not implemented for profile type %v", profileType))
}

type tagTaskCreator struct {
	profileID    uint64
	profileValue *string
	tagRepo      repo.TagRepo
}

func (c *tagTaskCreator) PreCreate(ctx context.Context, task *entity.Task) (*entity.Task, error) {
	tag, err := c.tagRepo.Get(ctx, &repo.TagFilter{
		ID:     goutil.Uint64(c.profileID),
		Status: goutil.Uint32(uint32(entity.TagStatusNormal)),
	})
	if err != nil {
		return nil, err
	}

	task.TagID = tag.ID

	if task.IsAdd() {
		if c.profileValue == nil {
			return nil, errors.New("add must have profile value")
		}
		// TODO: validate tag value type and enum
		task.TagValue = c.profileValue
	} else if task.IsDelete() {
		task.TagValue = nil
	}

	return task, nil
}
