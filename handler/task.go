package handler

import (
	"bufio"
	"bytes"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/router"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"net/mail"
	"strings"
	"sync"
	"time"
)

const CK_INSERT_BATCH_SIZE = 10_000

var queue = make(chan *entity.Task, 20)

var startTaskOnce sync.Once

type TaskHandler interface {
	CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error
}

type taskHandler struct {
	fileRepo         repo.FileRepo
	taskRepo         repo.TaskRepo
	tagRepo          repo.TagRepo
	queryRepo        repo.QueryRepo
	mappingIDHandler MappingIDHandler
}

func NewTaskHandler(fileRepo repo.FileRepo, taskRepo repo.TaskRepo,
	tagRepo repo.TagRepo, queryRepo repo.QueryRepo, mappingIDHandler MappingIDHandler) TaskHandler {

	h := &taskHandler{
		fileRepo:         fileRepo,
		taskRepo:         taskRepo,
		tagRepo:          tagRepo,
		queryRepo:        queryRepo,
		mappingIDHandler: mappingIDHandler,
	}

	h.newTaskProcessor(10)

	return h
}

func (h *taskHandler) newTaskProcessor(concurrency uint32) {
	startTaskOnce.Do(func() {
		go func() {
			var (
				g = new(errgroup.Group)
				c = make(chan struct{}, concurrency)
			)
			for {
				select {
				case task := <-queue:
					c <- struct{}{}

					func() {
						defer func() {
							<-c
						}()

						g.Go(func() error {
							if err := h.processTask(task); err != nil {
								return err
							}
							return nil
						})
					}()
				}
			}
		}()
	})
}

func (h *taskHandler) processTask(task *entity.Task) error {
	var (
		err   error
		logID = uuid.New()
		ctx   = log.With().Str("log_id", logID.String()).Logger().WithContext(context.Background())
	)

	log.Ctx(ctx).Info().Msgf("processing task: %v", task.GetID())

	taskFilter := &repo.TaskFilter{
		ID: task.ID,
	}
	defer func() {
		taskStatus := entity.TaskStatusSuccess
		if err != nil {
			taskStatus = entity.TaskStatusFailed
		}

		log.Ctx(ctx).Info().Msgf("task done! taskID: %v, task status: %v", task.GetID(), taskStatus)

		if err = h.taskRepo.Update(ctx, taskFilter, &entity.Task{
			Status: goutil.Uint32(uint32(taskStatus)),
		}); err != nil {
			log.Ctx(ctx).Error().Msgf("update task status failed: %v, status: %v", err, taskStatus)
		}
	}()

	if !task.IsPending() {
		log.Ctx(ctx).Info().Msg("task is not pending")
		return nil
	}

	tag, err := h.tagRepo.Get(ctx, &repo.TagFilter{
		ID: task.TagID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tag err: %v", err)
		return err
	}

	// download file
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

	var (
		scanner = bufio.NewScanner(bytes.NewReader(b))
		udIDs   = make([]string, 0)
		udTags  = new(entity.UdTags)
	)
	// process file
	for scanner.Scan() {
		var (
			line  = scanner.Text()
			parts = strings.Split(line, ",")
		)

		if len(parts) == 0 || len(parts) > 2 {
			log.Ctx(ctx).Error().Msgf("invalid line: %v", line)
			continue
		}

		// check if valid email
		udID := parts[0]
		if _, err = mail.ParseAddress(udID); err != nil {
			log.Ctx(ctx).Error().Msgf("invalid email address: %v", udID)
			continue
		}

		// check tag value
		var tagValue *string
		if len(parts) == 2 {
			value := parts[1]
			if value != "" {
				if ok := tag.IsValidTagValue(value); !ok {
					log.Ctx(ctx).Error().Msgf("invalid tag value, udID: %v, value: %v", udID, value)
					continue
				}
				tagValue = goutil.String(value)
			}
		}

		// no mapping ID yet
		udTag := &entity.UdTag{
			MappingID: &entity.MappingID{
				UdID: goutil.String(udID),
			},
			TagValue: tagValue,
		}

		udIDs = append(udIDs, udID)
		udTags.Data = append(udTags.Data, udTag)
	}

	// fetch mapping IDs
	mappingIDs := make(map[string]*entity.MappingID)
	for i := 0; i < len(udIDs); i += MappingIDsMaxSize {
		end := i + MappingIDsMaxSize
		if end > len(udIDs) {
			end = len(udIDs)
		}

		req := &GetSetMappingIDsRequest{
			UdIDs: udIDs[i:end],
		}
		res := new(GetSetMappingIDsResponse)

		if err = h.mappingIDHandler.GetSetMappingIDs(ctx, req, res); err != nil {
			log.Ctx(ctx).Error().Msgf("get set mapping ids failed: %v", err)
			return err
		}

		for _, mappingID := range res.MappingIDs {
			mappingIDs[mappingID.GetUdID()] = mappingID
		}
	}

	// insert into CK
	for i := 0; i < len(udTags.Data); i += CK_INSERT_BATCH_SIZE {
		end := i + CK_INSERT_BATCH_SIZE
		if end > len(udTags.Data) {
			end = len(udTags.Data)
		}

		subUdTags := &entity.UdTags{
			Tag:  tag,
			Data: udTags.Data[i:end],
		}

		// set mapping ID
		for _, udTag := range subUdTags.Data {
			udTag.MappingID = mappingIDs[udTag.MappingID.GetUdID()]
		}

		if err = h.queryRepo.Insert(ctx, subUdTags); err != nil {
			log.Ctx(ctx).Error().Msgf("failed to insert udTags into CK: %v", err)
			return err
		}
	}

	return nil
}

type CreateFileUploadTaskRequest struct {
	TagID *uint64 `schema:"tag_id,omitempty"`

	*router.FileMeta `json:"file_meta,omitempty"`
}

func (req *CreateFileUploadTaskRequest) GetTagID() uint64 {
	if req != nil && req.TagID != nil {
		return *req.TagID
	}
	return 0
}

type CreateFileUploadTaskResponse struct {
	Task *entity.Task `json:"task,omitempty"`
}

var CreateFileUploadTaskValidator = validator.MustForm(map[string]validator.Validator{
	"tag_id":    &validator.UInt64{},
	"file_info": FileInfoValidator(false, 5<<24, []string{"text/plain", "text/csv"}),
})

func (h *taskHandler) CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error {
	err := CreateFileUploadTaskValidator.Validate(req)
	if err != nil {
		return errutil.ValidationError(err)
	}

	_, err = h.tagRepo.Get(ctx, &repo.TagFilter{
		ID:     req.TagID,
		Status: goutil.Uint32(uint32(entity.TagStatusNormal)),
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tag failed: %v", err)
		return err
	}

	var (
		fileName = req.FileHeader.Filename
		fileKey  = h.generateFileKey(fileName)
	)
	url, err := h.fileRepo.Upload(ctx, fileKey, req.File)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("upload file %s failed, err: %v", fileName, err)
		return err
	}

	now := time.Now().Unix()
	task := &entity.Task{
		TagID:      req.TagID,
		FileName:   goutil.String(req.FileHeader.Filename),
		FileKey:    goutil.String(h.generateFileKey(req.FileHeader.Filename)),
		Status:     goutil.Uint32(uint32(entity.TaskStatusPending)),
		URL:        goutil.String(url),
		CreateTime: goutil.Uint64(uint64(now)),
		UpdateTime: goutil.Uint64(uint64(now)),
	}

	id, err := h.taskRepo.Create(ctx, task)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create task failed: %v", err)
		return err
	}
	task.ID = goutil.Uint64(id)

	go func() {
		select {
		case queue <- task:
			log.Ctx(context.Background()).Info().Msgf("task sent for processing, taskID: %v", task.GetID())
		}
	}()

	res.Task = task

	return nil
}

func (h *taskHandler) generateFileKey(fileName string) string {
	hashKey := fmt.Sprintf("%s-%d", fileName, time.Now().Unix())

	hFn := md5.New()
	hFn.Write([]byte(hashKey))
	hashValue := hFn.Sum(nil)

	return fmt.Sprintf("f-%s", hex.EncodeToString(hashValue))
}
