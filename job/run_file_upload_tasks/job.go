package run_file_upload_tasks

import (
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"fmt"
)

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

func (j *RunFileUploadTask) Init(ctx context.Context) error {
	fmt.Println("RunFileUploadTask Init")
	return nil
}

func (j *RunFileUploadTask) Run(ctx context.Context) error {
	fmt.Println("RunFileUploadTask Run")
	return nil
}

func (j *RunFileUploadTask) CleanUp(ctx context.Context) error {
	fmt.Println("RunFileUploadTask CleanUp")
	return nil
}
