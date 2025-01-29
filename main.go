package main

import (
	"cdp/config"
	"cdp/dep"
	"cdp/handler"
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
	baseRepo        repo.BaseRepo
	tagRepo         repo.TagRepo
	segmentRepo     repo.SegmentRepo
	fileRepo        repo.FileRepo
	mappingIDRepo   repo.MappingIDRepo
	emailRepo       repo.EmailRepo
	campaignRepo    repo.CampaignRepo
	campaignLogRepo repo.CampaignLogRepo
	tenantRepo      repo.TenantRepo
	userRepo        repo.UserRepo
	activationRepo  repo.ActivationRepo
	sessionRepo     repo.SessionRepo

	// services
	emailService dep.EmailService

	// api handlers
	tagHandler       handler.TagHandler
	segmentHandler   handler.SegmentHandler
	mappingIDHandler handler.MappingIDHandler
	emailHandler     handler.EmailHandler
	campaignHandler  handler.CampaignHandler
	tenantHandler    handler.TenantHandler
	userHandler      handler.UserHandler
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

	// base repo
	s.baseRepo, err = repo.NewBaseRepo(s.ctx, s.cfg.MetadataDB)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init base repo failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.baseRepo != nil {
			if err := s.baseRepo.Close(s.ctx); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close base repo failed, err: %v", err)
				return
			}
		}
	}()

	// segment repo
	s.segmentRepo = repo.NewSegmentRepo(s.ctx, s.baseRepo)

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
	s.emailRepo = repo.NewEmailRepo(s.ctx, s.baseRepo)

	// campaign repo
	s.campaignRepo = repo.NewCampaignRepo(s.ctx, s.baseRepo)

	// campaign log repo
	s.campaignLogRepo = repo.NewCampaignLogRepo(s.ctx, s.baseRepo)

	// tenant repo
	s.tenantRepo = repo.NewTenantRepo(s.ctx, s.baseRepo)

	// user repo
	s.userRepo = repo.NewUserRepo(s.ctx, s.baseRepo)

	// activation repo
	s.activationRepo = repo.NewActivationRepo(s.ctx, s.baseRepo)

	// session repo
	s.sessionRepo = repo.NewSessionRepo(s.ctx, s.baseRepo)

	// tag repo
	s.tagRepo = repo.NewTagRepo(s.ctx, s.baseRepo)

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
	s.segmentHandler = handler.NewSegmentHandler(s.cfg, s.tagRepo, s.segmentRepo)
	s.mappingIDHandler = handler.NewMappingIDHandler(s.mappingIDRepo)
	s.emailHandler = handler.NewEmailHandler(s.emailRepo)
	s.campaignHandler = handler.NewCampaignHandler(s.cfg, s.campaignRepo, s.emailService, s.segmentHandler, s.campaignLogRepo, s.emailHandler)
	s.tenantHandler = handler.NewTenantHandler(s.cfg, s.baseRepo, s.tenantRepo, s.userRepo, s.activationRepo, s.emailService)
	s.userHandler = handler.NewUserHandler(s.userRepo, s.tenantRepo, s.activationRepo, s.sessionRepo)

	// ===== start server ===== //

	go func() {
		addr := fmt.Sprintf(":%d", s.opt.Port)

		log.Info().Msgf("starting HTTP server at %s", addr)

		c := cors.New(cors.Options{
			AllowedOrigins:   []string{s.cfg.WebPage.Domain},
			AllowCredentials: true,
		})

		httpServer := &http.Server{
			BaseContext: func(_ net.Listener) context.Context {
				return s.ctx
			},
			Addr:    addr,
			Handler: router.Log(c.Handler(s.registerRoutes())),
		}
		err := httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Msgf("fail to start HTTP server, err: %v", err)
		}
	}()

	return nil
}

func (s *server) Stop() error {
	if s.baseRepo != nil {
		if err := s.baseRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close base repo failed, err: %v", err)
			return err
		}
	}

	if s.fileRepo != nil {
		if err := s.fileRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close entity file repo failed, err: %v", err)
			return err
		}
	}

	if s.mappingIDRepo != nil {
		if err := s.mappingIDRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close mapping id repo failed, err: %v", err)
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

type IsLoggedInRequest struct{}

type IsLoggedInResponse struct{}

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

	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathIsLoggedIn,
		Method: http.MethodGet,
		Handler: router.Handler{
			Req: new(IsLoggedInRequest),
			Res: new(IsLoggedInResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return nil
			},
		},
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
		},
	})

	// get_tag
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetTag,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetTagRequest),
			Res: new(handler.GetTagResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tagHandler.GetTag(ctx, req.(*handler.GetTagRequest), res.(*handler.GetTagResponse))
			},
		},
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
		},
	})

	// get_email
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetEmail,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetEmailRequest),
			Res: new(handler.GetEmailResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.emailHandler.GetEmail(ctx, req.(*handler.GetEmailRequest), res.(*handler.GetEmailResponse))
			},
		},
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
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
		Middlewares: []router.Middleware{
			router.NewSessionMiddleware(s.userRepo, s.tenantRepo, s.sessionRepo),
		},
	})

	// create_tenant
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateTenant,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateTenantRequest),
			Res: new(handler.CreateTenantResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tenantHandler.CreateTenant(ctx, req.(*handler.CreateTenantRequest), res.(*handler.CreateTenantResponse))
			},
		},
	})

	// get_tenant
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathGetTenant,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.GetTenantRequest),
			Res: new(handler.GetTenantResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tenantHandler.GetTenant(ctx, req.(*handler.GetTenantRequest), res.(*handler.GetTenantResponse))
			},
		},
	})

	// create_user
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathCreateUser,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.CreateUserRequest),
			Res: new(handler.CreateUserResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.userHandler.CreateUser(ctx, req.(*handler.CreateUserRequest), res.(*handler.CreateUserResponse))
			},
		},
	})

	// is_user_pending_init
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathIsUserPendingInit,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.IsUserPendingInitRequest),
			Res: new(handler.IsUserPendingInitResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.userHandler.IsUserPendingInit(ctx, req.(*handler.IsUserPendingInitRequest), res.(*handler.IsUserPendingInitResponse))
			},
		},
	})

	// init_user
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathInitUser,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.InitUserRequest),
			Res: new(handler.InitUserResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.userHandler.InitUser(ctx, req.(*handler.InitUserRequest), res.(*handler.InitUserResponse))
			},
		},
	})

	// init_tenant
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathInitTenant,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.InitTenantRequest),
			Res: new(handler.InitTenantResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tenantHandler.InitTenant(ctx, req.(*handler.InitTenantRequest), res.(*handler.InitTenantResponse))
			},
		},
	})

	// is_tenant_pending_activation
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathIsTenantPendingInit,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.IsTenantPendingInitRequest),
			Res: new(handler.IsTenantPendingInitResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.tenantHandler.IsTenantPendingInit(ctx, req.(*handler.IsTenantPendingInitRequest), res.(*handler.IsTenantPendingInitResponse))
			},
		},
	})

	// log_in
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathLogIn,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.LogInRequest),
			Res: new(handler.LogInResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.userHandler.LogIn(ctx, req.(*handler.LogInRequest), res.(*handler.LogInResponse))
			},
		},
	})

	// log_out
	r.RegisterHttpRoute(&router.HttpRoute{
		Path:   config.PathLogOut,
		Method: http.MethodPost,
		Handler: router.Handler{
			Req: new(handler.LogOutRequest),
			Res: new(handler.LogOutResponse),
			HandleFunc: func(ctx context.Context, req, res interface{}) error {
				return s.userHandler.LogOut(ctx, req.(*handler.LogOutRequest), res.(*handler.LogOutResponse))
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
