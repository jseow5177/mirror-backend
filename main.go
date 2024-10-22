package main

import (
	"cdp/config"
	"cdp/handler"
	"cdp/middleware"
	"cdp/pkg/mq"
	"cdp/pkg/router"
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
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

	tagRepo       repo.TagRepo
	fileRepo      repo.FileRepo
	taskRepo      repo.TaskRepo
	mappingIDRepo repo.MappingIDRepo
	queryRepo     repo.QueryRepo

	producer  *mq.Producer
	consumers []*mq.Consumer

	// api handlers
	tagHandler       handler.TagHandler
	taskHandler      handler.TaskHandler
	mappingIDHandler handler.MappingIDHandler
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

	// profile repo
	profileRepo := repo.NewProfileRepo(s.tagRepo)

	// query repo
	s.queryRepo, err = repo.NewQueryRepo(s.ctx, s.cfg.QueryDB)
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

	// ===== init consumers ===== //

	for i, cfg := range s.cfg.Consumers {
		var c *mq.Consumer
		c, err = mq.NewConsumer(s.ctx, cfg)
		if err != nil {
			log.Ctx(s.ctx).Error().Msgf("init consumer failed, idx: %v, err: %v", i, err)
			for i, c := range s.consumers {
				if err := c.Close(); err != nil {
					log.Ctx(s.ctx).Error().Msgf("close consumer failed, idx: %v, err: %v", i, err)
				}
			}
			return err
		}
		s.consumers = append(s.consumers, c)
	}
	defer func() {
		if err != nil {
			for i, c := range s.consumers {
				if err := c.Close(); err != nil {
					log.Ctx(s.ctx).Error().Msgf("close consumer failed, idx: %v, err: %v", i, err)
				}
			}
		}
	}()

	// ===== init mq producer ===== //

	s.producer, err = mq.NewProducer(s.ctx, s.cfg.Producer)
	if err != nil {
		log.Ctx(s.ctx).Error().Msgf("init producer failed, err: %v", err)
		return err
	}
	defer func() {
		if err != nil && s.producer != nil {
			if err := s.producer.Close(); err != nil {
				log.Ctx(s.ctx).Error().Msgf("close mq producer failed, err: %v", err)
				return
			}
		}
	}()

	// ===== init handlers ===== //

	s.tagHandler = handler.NewTagHandler(s.tagRepo)
	s.mappingIDHandler = handler.NewMappingIDHandler(s.mappingIDRepo)
	s.taskHandler = handler.NewTaskHandler(s.fileRepo, s.taskRepo, profileRepo, s.queryRepo, s.mappingIDHandler, s.producer)

	// ===== start server ===== //

	s.registerMQHandlers()

	go func() {
		addr := fmt.Sprintf(":%d", s.opt.Port)

		log.Info().Msgf("starting HTTP server at %s", addr)

		httpServer := &http.Server{
			BaseContext: func(_ net.Listener) context.Context {
				return s.ctx
			},
			Addr:    addr,
			Handler: middleware.Log(s.registerRoutes()),
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

	if s.queryRepo != nil {
		if err := s.queryRepo.Close(s.ctx); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close query repo failed, err: %v", err)
			return err
		}
	}

	for i, c := range s.consumers {
		if err := c.Close(); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close consumer failed, idx: %v, err: %v", i, err)
		}
	}

	if s.producer != nil {
		if err := s.producer.Close(); err != nil {
			log.Ctx(s.ctx).Error().Msgf("close mq producer failed, err: %v", err)
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
			ContentType: router.ContentTypeJSON,
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
			ContentType: router.ContentTypeFile,
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
			ContentType: router.ContentTypeJSON,
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
			ContentType: router.ContentTypeJSON,
		},
	})

	return r
}

func (s *server) registerMQHandlers() {
	mq.RegisterHandler(mq.PayloadNotifyCreateTask, func(ctx context.Context, msg *mq.Message) error {
		req := new(mq.NotifyCreateTask)
		if err := msg.ParseBody(req); err != nil {
			return err
		}
		return s.taskHandler.WriteFileToStorage(ctx, req)
	})
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
