package main

import (
	"cdp/config"
	"cdp/dep"
	"cdp/handler"
	"cdp/job/hello_world"
	"cdp/job/run_campaigns"
	"cdp/job/run_file_upload_tasks"
	"cdp/pkg/logutil"
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"github.com/rs/zerolog/log"
	"os"
)

func main() {
	var (
		opt = config.NewOptions()
		ctx = logutil.InitZeroLog(context.Background(), opt.LogLevel)
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

	emailService, err := dep.NewEmailService(ctx, cfg.SMTP)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("init email service failed, err: %v", err)
		os.Exit(1)
	}
	defer func() {
		if err != nil && emailService != nil {
			if err := emailService.Close(ctx); err != nil {
				log.Ctx(ctx).Error().Msgf("close email service failed, err: %v", err)
				return
			}
		}
	}()

	// tag repo
	tagRepo := repo.NewTagRepo(ctx, baseRepo)

	// task repo
	taskRepo := repo.NewTaskRepo(ctx, baseRepo)

	// tenant repo
	tenantRepo, err := repo.NewTenantRepo(ctx, baseRepo)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("init tenant repo failed, err: %v", err)
		os.Exit(1)
	}

	// campaign repo
	campaignRepo := repo.NewCampaignRepo(ctx, baseRepo)

	// segment repo
	segmentRepo := repo.NewSegmentRepo(ctx, baseRepo)

	// email repo
	emailRepo := repo.NewEmailRepo(ctx, baseRepo)

	// sender repo
	senderRepo := repo.NewSenderRepo(ctx, baseRepo)

	// segment handler
	segmentHandler := handler.NewSegmentHandler(cfg, tagRepo, segmentRepo, queryRepo)

	// email handler
	emailHandler := handler.NewEmailHandler(emailRepo)

	jobs := map[string]service.Job{
		"hello-world":           hello_world.New(),
		"run-file-upload-tasks": run_file_upload_tasks.New(taskRepo, fileRepo, queryRepo, tenantRepo, tagRepo),
		"run-campaigns": run_campaigns.New(cfg, campaignRepo, emailService, segmentHandler,
			emailHandler, tenantRepo, senderRepo),
	}

	if len(os.Args) < 2 {
		log.Ctx(ctx).Error().Msg("Usage: go run main.go <job_name>")
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

	if baseRepo != nil {
		if err := baseRepo.Close(ctx); err != nil {
			log.Ctx(ctx).Error().Msgf("close base repo failed, err: %v", err)
		}
	}

	if fileRepo != nil {
		if err := fileRepo.Close(ctx); err != nil {
			log.Ctx(ctx).Error().Msgf("close entity file repo failed, err: %v", err)
		}
	}

	if emailService != nil {
		if err := emailService.Close(ctx); err != nil {
			log.Ctx(ctx).Error().Msgf("close email service failed, err: %v", err)
		}
	}

	if queryRepo != nil {
		if err := queryRepo.Close(ctx); err != nil {
			log.Ctx(ctx).Error().Msgf("close query repo failed, err: %v", err)
		}
	}

	log.Ctx(ctx).Info().Msg("job executed successfully")
	os.Exit(0)
}
