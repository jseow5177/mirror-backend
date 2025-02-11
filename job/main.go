package main

import (
	"cdp/config"
	"cdp/job/hello_world"
	"cdp/job/run_file_upload_tasks"
	"cdp/pkg/logutil"
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"os"
)

func main() {
	var (
		opt = config.NewOptions()
		ctx = logutil.InitZeroLog(context.Background(), "DEBUG")
	)

	cfg := config.NewConfig()
	if err := cfg.Load(ctx, opt.ConfigPath); err != nil {
		log.Ctx(ctx).Error().Msgf("load config failed: %v", err)
		os.Exit(1)
	}

	// base repo
	baseRepo, err := repo.NewBaseRepo(ctx, cfg.MetadataDB)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("init base repo failed, err: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err != nil && baseRepo != nil {
			if err := baseRepo.Close(ctx); err != nil {
				log.Ctx(ctx).Error().Msgf("close base repo failed, err: %v", err)
				return
			}
		}
	}()

	// query repo
	queryRepo, err := repo.NewQueryRepo(ctx, cfg.QueryDB)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("init query repo failed, err: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err != nil && queryRepo != nil {
			if err := queryRepo.Close(ctx); err != nil {
				log.Ctx(ctx).Error().Msgf("close query repo failed, err: %v", err)
				return
			}
		}
	}()

	// file repo
	fileRepo, err := repo.NewFileRepo(ctx, cfg.FileStore)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("init file repo failed, err: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err != nil && fileRepo != nil {
			if err := fileRepo.Close(ctx); err != nil {
				log.Ctx(ctx).Error().Msgf("close file repo failed, err: %v", err)
				return
			}
		}
	}()

	// tag repo
	tagRepo := repo.NewTagRepo(ctx, baseRepo)

	// task repo
	taskRepo := repo.NewTaskRepo(ctx, baseRepo)

	// tenant repo
	tenantRepo := repo.NewTenantRepo(ctx, baseRepo)

	jobs := map[string]service.Job{
		"hello-world":           hello_world.New(),
		"run-file-upload-tasks": run_file_upload_tasks.New(taskRepo, fileRepo, queryRepo, tenantRepo, tagRepo),
	}

	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <job_name>")
		os.Exit(1)
	}

	jobName := os.Args[1]
	job, exists := jobs[jobName]
	if !exists {
		log.Ctx(ctx).Error().Msgf("job %s not found", jobName)
		os.Exit(1)
	}

	if err := job.Init(ctx); err != nil {
		log.Ctx(ctx).Error().Msgf("init job err: %v", err)
		os.Exit(1)
	}

	if err := job.Run(ctx); err != nil {
		log.Ctx(ctx).Error().Msgf("run job err: %v", err)
		os.Exit(1)
	}

	if err := job.CleanUp(ctx); err != nil {
		log.Ctx(ctx).Error().Msgf("cleanup job err: %v", err)
		os.Exit(1)
	}

	log.Ctx(ctx).Info().Msg("job executed successfully")
	os.Exit(0)
}
