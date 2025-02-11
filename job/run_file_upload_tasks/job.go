package run_file_upload_tasks

import (
	"cdp/entity"
	"cdp/pkg/goutil"
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"time"
)

const batchSize = 3_000

type RunFileUploadTask struct {
	taskRepo   repo.TaskRepo
	fileRepo   repo.FileRepo
	queryRepo  repo.QueryRepo
	tenantRepo repo.TenantRepo
	tagRepo    repo.TagRepo
}

func New(taskRepo repo.TaskRepo, fileRepo repo.FileRepo, queryRepo repo.QueryRepo,
	tenantRepo repo.TenantRepo, tagRepo repo.TagRepo) service.Job {
	return &RunFileUploadTask{
		taskRepo:   taskRepo,
		fileRepo:   fileRepo,
		queryRepo:  queryRepo,
		tenantRepo: tenantRepo,
		tagRepo:    tagRepo,
	}
}

func (h *RunFileUploadTask) Init(_ context.Context) error {
	return nil
}

func (h *RunFileUploadTask) Run(ctx context.Context) error {
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

	log.Ctx(ctx).Info().Msgf("number of tasks to be processed: %d", len(tasks))

	type taskStatus struct {
		err    error
		task   *entity.Task
		status entity.TaskStatus
	}

	// track task status
	var (
		statusChan       = make(chan taskStatus, len(tasks))
		updateTaskStatus = func(status entity.TaskStatus, task *entity.Task, err error) {
			statusChan <- taskStatus{err: err, task: task, status: status}
		}
	)
	g.Go(func() error {
		for {
			select {
			case te := <-statusChan:
				task := te.task
				if te.err != nil {
					log.Ctx(ctx).Error().Msgf("[task ID %d] error encountered: %v", task.GetID(), te.err)
				}

				task.Update(&entity.Task{
					Status: te.status,
				})
				if err = h.taskRepo.Update(ctx, task); err != nil {
					log.Ctx(ctx).Error().Msgf("[task ID %d] set campaign status failed err: %v, status: %v", task.GetID(), err, te.status)
				}
			}
		}
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
				updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("get tenant failed: %v", err))
				return err
			}

			// get tag
			tag, err := h.tagRepo.GetByID(ctx, tenant.GetID(), task.GetResourceID())
			if err != nil {
				updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("get tag failed: %v", err))
				return err
			}

			// download file data
			rows, err := h.fileRepo.DownloadFile(ctx, task.GetFileID())
			if err != nil {
				updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("get file %s failed: %v", fileID, err))
				return err
			}

			// set task to running
			task.Update(&entity.Task{
				Status: entity.TaskStatusRunning,
			})
			if err := h.taskRepo.Update(ctx, task); err != nil {
				updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("set task to running failed: %v", err))
				return err
			}

			// split data into batches
			var (
				batches = make([][]*entity.UdTagVal, 0)
				batch   = make([]*entity.UdTagVal, 0)
			)
			for i, row := range rows {
				if len(row) != 2 {
					updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("invalid row: %v, file: %s", row, fileID))
					return err
				}

				v, err := tag.FormatTagValue(row[1])
				if err != nil {
					updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("invalid tag value: %v, file: %s", rows[1], fileID))
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

			// start to write
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
							updateTaskStatus(entity.TaskStatusRunning, task, fmt.Errorf("set task progress err: %v", err))
						}
					}
				default:
				}

				// batch upsert
				if batchNum < len(batches) {
					batch = batches[batchNum]

					if err := h.queryRepo.BatchUpsert(ctx, tenant.GetName(), batch); err != nil {
						updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("batch upsert err: %v", err))
						return err
					}

					batchNum++
				}

				// full completion
				if count == task.GetSize() {
					task.Update(&entity.Task{
						Status: entity.TaskStatusSuccess,
						ExtInfo: &entity.TaskExtInfo{
							Progress: goutil.Uint64(100),
						},
					})
					log.Ctx(ctx).Info().Msgf("task is success, task_id: %v", task.GetID())
					if err := h.taskRepo.Update(ctx, task); err != nil {
						updateTaskStatus(entity.TaskStatusFailed, task, fmt.Errorf("set task to 100%% completion err: %v", err))
						return err
					}

					// done
					return nil
				}
			}
		})
	}

	return nil
}

func (h *RunFileUploadTask) CleanUp(_ context.Context) error {
	return nil
}
