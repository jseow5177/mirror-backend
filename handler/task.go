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
	"golang.org/x/sync/errgroup"
	"mime/multipart"
	"time"
)

const batchSize = 3_000

type TaskHandler interface {
	CreateFileUploadTask(ctx context.Context, req *CreateFileUploadTaskRequest, res *CreateFileUploadTaskResponse) error
	GetFileUploadTasks(ctx context.Context, req *GetFileUploadTasksRequest, res *GetFileUploadTasksResponse) error
	RunFileUploadTasks(ctx context.Context, req *RunFileUploadTasksRequest, res *RunFileUploadTasksResponse) error
}

type taskHandler struct {
	taskRepo   repo.TaskRepo
	fileRepo   repo.FileRepo
	queryRepo  repo.QueryRepo
	tenantRepo repo.TenantRepo
	tagRepo    repo.TagRepo
}

func NewTaskHandler(taskRepo repo.TaskRepo, fileRepo repo.FileRepo, queryRepo repo.QueryRepo, tenantRepo repo.TenantRepo, tagRepo repo.TagRepo) TaskHandler {
	return &taskHandler{
		taskRepo,
		fileRepo,
		queryRepo,
		tenantRepo,
		tagRepo,
	}
}

type RunFileUploadTasksRequest struct{}

type RunFileUploadTasksResponse struct{}

func (h *taskHandler) RunFileUploadTasks(ctx context.Context, _ *RunFileUploadTasksRequest, _ *RunFileUploadTasksResponse) error {
	ctx = context.WithoutCancel(ctx)

	var (
		g  = new(errgroup.Group)
		c  = 10
		ch = make(chan struct{}, c)
	)

	// get tag resource only
	tasks, err := h.taskRepo.GetPendingFileUploadTasks(ctx, entity.ResourceTypeTag)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get pending file upload tasks failed: %v", err)
		return err
	}

	type taskStatus struct {
		err  error
		task *entity.Task
	}

	// track task status
	statusChan := make(chan taskStatus, len(tasks))
	g.Go(func() error {
		var count int
		for {
			select {
			case te := <-statusChan:
				var (
					task   = te.task
					status = entity.TaskStatusSuccess
				)
				if te.err != nil {
					status = entity.TaskStatusFailed
					log.Ctx(ctx).Error().Msgf("[task ID %d] error encountered: %v", task.GetID(), te.err)
				} else {
					log.Ctx(ctx).Info().Msgf("[task ID %d]: task success!", task.GetID())
				}

				task.Update(&entity.Task{
					Status: status,
				})
				if err = h.taskRepo.Update(ctx, task); err != nil {
					log.Ctx(ctx).Error().Msgf("[task ID %d] set campaign status failed err: %v, status: %v", task.GetID(), err, status)
				}
			}

			count++
			if count == len(tasks) {
				break
			}
		}
		return nil
	})

	// process tasks
	for _, task := range tasks {
		select {
		case ch <- struct{}{}:
		}

		task := task
		g.Go(func() error {
			// release go routine
			defer func() {
				<-ch
			}()

			fileID := task.GetFileID()

			// get tenant
			tenant, err := h.tenantRepo.GetByID(ctx, task.GetTenantID())
			if err != nil {
				statusChan <- taskStatus{
					err:  fmt.Errorf("get tenant failed: %v", err),
					task: task,
				}
				return err
			}

			// get tag
			tag, err := h.tagRepo.GetByID(ctx, tenant.GetID(), task.GetResourceID())
			if err != nil {
				statusChan <- taskStatus{
					err:  fmt.Errorf("get tag failed: %v", err),
					task: task,
				}
				return err
			}

			// download file data
			rows, err := h.fileRepo.DownloadFile(ctx, task.GetFileID())
			if err != nil {
				statusChan <- taskStatus{
					err:  fmt.Errorf("get file %s failed: %v", fileID, err),
					task: task,
				}
				return err
			}

			// set task to running
			task.Update(&entity.Task{
				Status: entity.TaskStatusRunning,
			})
			if err := h.taskRepo.Update(ctx, task); err != nil {
				statusChan <- taskStatus{
					err:  fmt.Errorf("set task to running failed: %v", err),
					task: task,
				}
				return err
			}

			// split data into batches
			var (
				batches = make([][]*entity.UdTagVal, 0)
				batch   = make([]*entity.UdTagVal, 0)
			)
			for i, row := range rows {
				if len(row) != 2 {
					statusChan <- taskStatus{
						err:  fmt.Errorf("invalid row: %v, file: %s", row, fileID),
						task: task,
					}
					return err
				}

				v, err := tag.FormatTagValue(row[1])
				if err != nil {
					statusChan <- taskStatus{
						err:  fmt.Errorf("invalid tag value: %v, file: %s", rows[1], fileID),
						task: task,
					}
					return err
				}

				batch = append(batch, &entity.UdTagVal{
					Ud: &entity.Ud{
						ID:     goutil.String(row[0]),
						IDType: entity.IDTypeEmail,
					},
					TagVals: []*entity.TagVal{
						{
							TagID:  task.ResourceID,
							TagVal: v,
						},
					},
				})

				if len(batch) >= batchSize || i == len(rows)-1 {
					batches = append(batches, batch)
					batch = make([]*entity.UdTagVal, 0)
				}
			}

			subG := new(errgroup.Group)
			subG.Go(func() error {
				var (
					count    uint64
					batchNum int
				)

				ticker := time.NewTicker(2 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-h.queryRepo.OnInsertSuccess():
						count++
					case insertErr := <-h.queryRepo.OnInsertFailure():
						count++
						return insertErr
					case <-ticker.C:
						if task.GetSize() > 0 {
							progress := count * 100 / task.GetSize()
							task.Update(&entity.Task{
								ExtInfo: &entity.TaskExtInfo{
									Progress: goutil.Uint64(progress),
								},
							})
							// no need return err, let the next update to correct the error
							if err := h.taskRepo.Update(ctx, task); err != nil {
								log.Ctx(ctx).Error().Msgf("set task progress err: %v, task_id: %v", err, task.GetID())
							}
						}
					default:
					}

					// batch upsert
					if batchNum < len(batches) {
						batch = batches[batchNum]

						if err := h.queryRepo.BatchUpsert(ctx, tenant.GetName(), batch); err != nil {
							return fmt.Errorf("batch upsert err: %v", err)
						}

						batchNum++
					}

					// full completion
					if count == task.GetSize() {
						task.Update(&entity.Task{
							ExtInfo: &entity.TaskExtInfo{
								Progress: goutil.Uint64(100),
							},
						})
						if err := h.taskRepo.Update(ctx, task); err != nil {
							return fmt.Errorf("set task to 100%% completion err: %v", err)
						}

						return nil
					}
				}
			})

			if err := subG.Wait(); err != nil {
				statusChan <- taskStatus{err: err, task: task}
				return err
			}

			statusChan <- taskStatus{err: nil, task: task}

			return nil
		})
	}

	return nil
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
		TenantID:     req.Tenant.ID,
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

	fileName := fmt.Sprintf("%s:%d",
		req.GetFileName(),
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

	_, err := file.Seek(0, 0)
	if err != nil {
		return 0, err
	}

	return count, nil
}
