package main

import (
	"cdp/config"
	"cdp/dep"
	"cdp/handler"
	"cdp/middleware"
	"cdp/pkg/router"
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/rs/zerolog"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

type server struct {
	ctx context.Context
	opt *config.Option
	cfg *config.Config

	// repos
	tagRepo           repo.TagRepo
	segmentRepo       repo.SegmentRepo
	fileRepo          repo.FileRepo
	taskRepo          repo.TaskRepo
	mappingIDRepo     repo.MappingIDRepo
	queryRepo         repo.QueryRepo
	emailRepo         repo.EmailRepo
	campaignRepo      repo.CampaignRepo
	campaignEmailRepo repo.CampaignEmailRepo
	campaignLogRepo   repo.CampaignLogRepo

	// services
	emailService dep.EmailService

	// api handlers
	tagHandler       handler.TagHandler
	segmentHandler   handler.SegmentHandler
	taskHandler      handler.TaskHandler
	mappingIDHandler handler.MappingIDHandler
	emailHandler     handler.EmailHandler
	campaignHandler  handler.CampaignHandler
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	s := new(server)
	if err := service.Run(s); err != nil {
		log.Fatal().Msg(err.Error())
	}
}

func (s *server) Init() error {
	opt := config.NewOptions()

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		opt.LogLevel = logLevel
	}

	if configPath := os.Getenv("CONFIG_PATH"); configPath != "" {
		opt.ConfigPath = configPath
	}

	if serverPort := os.Getenv("PORT"); serverPort != "" {
		if port, err := strconv.Atoi(serverPort); err == nil {
			opt.Port = port
		}
	}

	s.opt = opt

	return nil
}

func (s *server) Start() error {
	var err error

	// ====== init logger ===== //

	s.ctx = initZeroLog(context.Background(), s.opt.LogLevel)

	// ===== init config ===== //

	s.cfg = config.NewConfig()
	if err = s.cfg.Load(s.ctx, s.opt.ConfigPath); err != nil {
		log.Ctx(s.ctx).Error().Msgf("load config failed, err: %v", err)
		return err
	}

	// ===== init repos =====

	// tag repo
	s.tagRepo, err = repo.NewTagRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init tag repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.tagRepo != nil {
			if err := s.tagRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close tag repo failed, err: %v", err)
				return
			}
		}
	}()

	s.segmentRepo, err = repo.NewSegmentRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init segment repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.segmentRepo != nil {
			if err := s.segmentRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close segment repo failed, err: %v", err)
				return
			}
		}
	}()

	// file repo
	s.fileRepo = repo.NewFileRepo(s.ctx, s.cfg.FileStore)
	defer func() {
		if err != nil && s.fileRepo != nil {
			if err := s.fileRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close file repo failed, err: %v", err)
				return
			}
		}
	}()

	// task repo
	s.taskRepo, err = repo.NewTaskRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init task repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.taskRepo != nil {
			if err := s.taskRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close task repo failed, err: %v", err)
				return
			}
		}
	}()

	// mapping ID repo
	s.mappingIDRepo, err = repo.NewMappingIDRepo(s.ctx, s.cfg.MappingIdDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init mapping id repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.mappingIDRepo != nil {
			if err := s.mappingIDRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close mapping id repo failed, err: %v", err)
				return
			}
		}
	}()

	// email repo
	s.emailRepo, err = repo.NewEmailRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init email repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.emailRepo != nil {
			if err := s.emailRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close email repo failed, err: %v", err)
				return
			}
		}
	}()

	// query repo
	s.queryRepo, err = repo.NewQueryRepo(s.ctx, s.cfg.QueryDB, s.tagRepo)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init query repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.queryRepo != nil {
			if err := s.queryRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close query repo failed, err: %v", err)
				return
			}
		}
	}()

	// campaign_email repo
	s.campaignEmailRepo, err = repo.NewCampaignEmailRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init campaign email repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.campaignEmailRepo != nil {
			if err := s.campaignEmailRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close campaign email repo failed, err: %v", err)
				return
			}
		}
	}()

	// campaign repo
	s.campaignRepo, err = repo.NewCampaignRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init campaign repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.campaignRepo != nil {
			if err := s.campaignRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close campaign repo failed, err: %v", err)
				return
			}
		}
	}()

	// campaign log repo
	s.campaignLogRepo, err = repo.NewCampaignLogRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init campaign log repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.campaignLogRepo != nil {
			log.Ctx(s.ctx).Error().Msgf("close campaign log repo failed, err: %v", err)
			return
		}
	}()

	// ===== init deps ===== //

	s.emailService, err = dep.NewEmailService(s.ctx, s.cfg)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init email service failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.emailService != nil {
			if err := s.emailService.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close email service failed, err: %v", err)
				return
			}
		}
	}()

	// ===== init handlers ===== //

	s.tagHandler = handler.NewTagHandler(s.tagRepo)
	s.segmentHandler = handler.NewSegmentHandler(s.cfg, s.tagRepo, s.segmentRepo, s.queryRepo)
	s.mappingIDHandler = handler.NewMappingIDHandler(s.mappingIDRepo)
	s.taskHandler = handler.NewTaskHandler(s.fileRepo, s.taskRepo, s.tagRepo, s.queryRepo, s.mappingIDHandler)
	s.emailHandler = handler.NewEmailHandler(s.emailRepo)
	s.campaignHandler = handler.NewCampaignHandler(s.cfg, s.campaignRepo, s.emailHandler, s.emailService, s.segmentHandler, s.campaignEmailRepo, s.campaignLogRepo)

	// ===== start server ===== //

	go func() {
		addr := fmt.Sprintf(":%d", s.opt.Port)

		log.Info().Msgf("starting HTTP server at %s", addr)

		httpServer := &http.Server{
			BaseContext: func(_ net.Listener) context.Context {
				return s.ctx
			},
			Addr:    addr,
			Handler: middleware.Log(cors.Default().Handler(s.registerRoutes())),
		}
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Msgf("fail to start HTTP server, err: %v", err)
		}
	}()

	return nil
}

func (s *server) Stop() error {
	if s.tagRepo != nil {
		if err := s.tagRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close tag repo failed, err: %v", err)
			return err
		}
	}

	if s.segmentRepo != nil {
		if err := s.segmentRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close segment repo failed, err: %v", err)
			return err
		}
	}

	if s.fileRepo != nil {
		if err := s.fileRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close entity file repo failed, err: %v", err)
			return err
		}
	}

	if s.taskRepo != nil {
		if err := s.taskRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close task repo failed, err: %v", err)
			return err
		}
	}

	if s.mappingIDRepo != nil {
		if err := s.mappingIDRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close mapping id repo failed, err: %v", err)
			return err
		}
	}

	if s.emailRepo != nil {
		if err := s.emailRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close email repo failed, err: %v", err)
			return err
		}
	}

	if s.queryRepo != nil {
		if err := s.queryRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close query repo failed, err: %v", err)
			return err
		}
	}

	if s.campaignRepo != nil {
		if err := s.campaignRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close campaign repo failed, err: %v", err)
			return err
		}
	}

	if s.campaignLogRepo != nil {
		if err := s.campaignLogRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close campaign log repo failed, err: %v", err)
			return err
		}
	}

	if s.emailService != nil {
		if err := s.emailService.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close email service failed, err: %v", err)
			return err
		}
	}

	return nil
}

type HealthCheckRequest struct{}

type HealthCheckResponse struct{}

func (s *server) registerRoutes() http.Handler {
	r := &router.HttpRouter{
		Router: mux.NewRouter(),
	}

	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathHealthCheck,
		Method: http.MethodGet,
		Handler: router.Handler{
			Req: new(HealthCheckRequest),
			Res: new(HealthCheckResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return nil
			},
		},
	})

	// get_tags
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetTags,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetTagsRequest),
			Res: new(handler.GetTagsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tagHandler.GetTags(ctx, req.(*handler.GetTagsRequest), res.(*handler.GetTagsResponse))
			},
		},
	})

	// get_segment
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetSegment,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetSegmentRequest),
			Res: new(handler.GetSegmentResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.segmentHandler.GetSegment(ctx, req.(*handler.GetSegmentRequest), res.(*handler.GetSegmentResponse))
			},
		},
	})

	// get_segments
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetSegments,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetSegmentsRequest),
			Res: new(handler.GetSegmentsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.segmentHandler.GetSegments(ctx, req.(*handler.GetSegmentsRequest), res.(*handler.GetSegmentsResponse))
			},
		},
	})

	// count_tags
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCountTags,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CountTagsRequest),
			Res: new(handler.CountTagsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tagHandler.CountTags(ctx, req.(*handler.CountTagsRequest), res.(*handler.CountTagsResponse))
			},
		},
	})

	// count_segments
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCountSegments,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CountSegmentsRequest),
			Res: new(handler.CountSegmentsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.segmentHandler.CountSegments(ctx, req.(*handler.CountSegmentsRequest), res.(*handler.CountSegmentsResponse))
			},
		},
	})

	// create_tag
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateTag,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateTagRequest),
			Res: new(handler.CreateTagResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tagHandler.CreateTag(ctx, req.(*handler.CreateTagRequest), res.(*handler.CreateTagResponse))
			},
		},
	})

	// create_segment
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateSegment,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateSegmentRequest),
			Res: new(handler.CreateSegmentResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.segmentHandler.CreateSegment(ctx, req.(*handler.CreateSegmentRequest), res.(*handler.CreateSegmentResponse))
			},
		},
	})

	// count_ud
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCountUd,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CountUdRequest),
			Res: new(handler.CountUdResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.segmentHandler.CountUd(ctx, req.(*handler.CountUdRequest), res.(*handler.CountUdResponse))
			},
		},
	})

	// preview_ud
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathPreviewUd,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.PreviewUdRequest),
			Res: new(handler.PreviewUdResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.segmentHandler.PreviewUd(ctx, req.(*handler.PreviewUdRequest), res.(*handler.PreviewUdResponse))
			},
		},
	})

	// upload_file
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateFileUploadTask,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateFileUploadTaskRequest),
			Res: new(handler.CreateFileUploadTaskResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.taskHandler.CreateFileUploadTask(ctx, req.(*handler.CreateFileUploadTaskRequest), res.(*handler.CreateFileUploadTaskResponse))
			},
		},
	})

	// get_mapping_ids
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetMappingIDs,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetMappingIDsRequest),
			Res: new(handler.GetMappingIDsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.mappingIDHandler.GetMappingIDs(ctx, req.(*handler.GetMappingIDsRequest), res.(*handler.GetMappingIDsResponse))
			},
		},
	})

	// get_set_mapping_ids
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetSetMappingIDs,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetSetMappingIDsRequest),
			Res: new(handler.GetSetMappingIDsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.mappingIDHandler.GetSetMappingIDs(ctx, req.(*handler.GetSetMappingIDsRequest), res.(*handler.GetSetMappingIDsResponse))
			},
		},
	})

	// create_email
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateEmail,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateEmailRequest),
			Res: new(handler.CreateEmailResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.emailHandler.CreateEmail(ctx, req.(*handler.CreateEmailRequest), res.(*handler.CreateEmailResponse))
			},
		},
	})

	// get_emails
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetEmails,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetEmailsRequest),
			Res: new(handler.GetEmailsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.emailHandler.GetEmails(ctx, req.(*handler.GetEmailsRequest), res.(*handler.GetEmailsResponse))
			},
		},
	})

	// create_campaign
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateCampaign,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateCampaignRequest),
			Res: new(handler.CreateCampaignResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.campaignHandler.CreateCampaign(ctx, req.(*handler.CreateCampaignRequest), res.(*handler.CreateCampaignResponse))
			},
		},
	})

	// run_campaigns
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathRunCampaigns,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.RunCampaignsRequest),
			Res: new(handler.RunCampaignsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.campaignHandler.RunCampaigns(ctx, req.(*handler.RunCampaignsRequest), res.(*handler.RunCampaignsResponse))
			},
		},
	})

	// on_email_action
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathOnEmailAction,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.OnEmailActionRequest),
			Res: new(handler.OnEmailActionResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.campaignHandler.OnEmailAction(ctx, req.(*handler.OnEmailActionRequest), res.(*handler.OnEmailActionResponse))
			},
		},
	})

	// get_campaigns
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetCampaigns,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetCampaignsRequest),
			Res: new(handler.GetCampaignsResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.campaignHandler.GetCampaigns(ctx, req.(*handler.GetCampaignsRequest), res.(*handler.GetCampaignsResponse))
			},
		},
	})

	// get_campaign
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetCampaign,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetCampaignRequest),
			Res: new(handler.GetCampaignResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.campaignHandler.GetCampaign(ctx, req.(*handler.GetCampaignRequest), res.(*handler.GetCampaignResponse))
			},
		},
	})

	return r
}

func initZeroLog(ctx context.Context, level string) context.Context {
	// use unix time
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// set log level
	var logLevel zerolog.Level
	switch strings.ToLower(level) {
	case zerolog.LevelDebugValue:
		logLevel = zerolog.DebugLevel
	case zerolog.LevelInfoValue:
		logLevel = zerolog.InfoLevel
	case zerolog.LevelWarnValue:
		logLevel = zerolog.WarnLevel
	case zerolog.LevelErrorValue:
		logLevel = zerolog.ErrorLevel
	case zerolog.LevelFatalValue:
		logLevel = zerolog.FatalLevel
	default:
		logLevel = zerolog.TraceLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// show caller: github.com/rs/zerolog#add-file-and-line-number-to-log
	zerolog.CallerMarshalFunc = func(_ uintptr, file string, line int) string {
		short := file
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				short = file[i+1:]
				break
			}
		}
		file = short
		return fmt.Sprintf("%s:%d", short, line)
	}
	log.Logger = log.With().Caller().Logger()

	ctx = log.Logger.WithContext(ctx)
	return ctx
}
