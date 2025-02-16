package handler

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

type AccountHandler interface {
	CreateTrialAccount(ctx context.Context, req *CreateTrialAccountRequest, res *CreateTrialAccountResponse) error
}

type accountHandler struct {
	cfg            *config.Config
	tenantHandler  TenantHandler
	userHandler    UserHandler
	tagHandler     TagHandler
	segmentHandler SegmentHandler
	emailHandler   EmailHandler
	campaignHandle CampaignHandler
	queryRepo      repo.QueryRepo
	taskRepo       repo.TaskRepo
}

func NewAccountHandler(cfg *config.Config, userHandler UserHandler, tenantHandler TenantHandler,
	tagHandler TagHandler, segmentHandler SegmentHandler, emailHandler EmailHandler, campaignHandler CampaignHandler,
	queryRepo repo.QueryRepo, taskRepo repo.TaskRepo) AccountHandler {
	return &accountHandler{
		cfg:            cfg,
		tenantHandler:  tenantHandler,
		userHandler:    userHandler,
		tagHandler:     tagHandler,
		segmentHandler: segmentHandler,
		emailHandler:   emailHandler,
		campaignHandle: campaignHandler,
		queryRepo:      queryRepo,
		taskRepo:       taskRepo,
	}
}

type CreateTrialAccountRequest struct {
	Token *string `schema:"token,omitempty"`
}

func (r *CreateTrialAccountRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type CreateTrialAccountResponse struct {
	Session *entity.Session `json:"session,omitempty"`
}

var CreateTrialAccountValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
})

func (h *accountHandler) CreateTrialAccount(ctx context.Context, req *CreateTrialAccountRequest, res *CreateTrialAccountResponse) error {
	if err := CreateTrialAccountValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.GetToken() != h.cfg.TrialAccountToken {
		return errutil.ValidationError(errors.New("invalid trial account token"))
	}

	var (
		suffix     = strings.ToLower(goutil.GenerateRandString(15))
		tenantName = fmt.Sprintf("demo-mirror-%s", suffix)
	)

	var (
		createTenantReq = &CreateTenantRequest{
			Name: goutil.String(tenantName),
		}
		createTenantRes = new(CreateTenantResponse)
	)
	if err := h.tenantHandler.CreateTenant(ctx, createTenantReq, createTenantRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create tenant error: %s", err)
		return err
	}

	password, err := goutil.GenerateSecureRandString(15)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("generate random password failed: %v", err)
		return err
	}

	var (
		username = "admin"
		email    = fmt.Sprintf("%s@mirror.com", username)

		initTenantReq = &InitTenantRequest{
			Token: createTenantRes.Token,
			User: &CreateUserRequest{
				Email:    goutil.String(email),
				Password: goutil.String(password),
			},
		}
		initTenantRes = new(InitTenantResponse)
	)
	if err := h.tenantHandler.InitTenant(ctx, initTenantReq, initTenantRes); err != nil {
		log.Ctx(ctx).Error().Msgf("init tenant error: %s", err)
		return err
	}

	var (
		tenant    = initTenantRes.Tenant
		adminUser = initTenantRes.Users[0]
	)

	contextInfo := ContextInfo{
		Tenant: tenant,
		User:   adminUser,
	}

	var (
		createTagReqs = []*CreateTagRequest{
			{
				ContextInfo: contextInfo,
				Name:        goutil.String("Age"),
				TagDesc:     goutil.String("Age of Users"),
				ValueType:   goutil.Uint32(uint32(entity.TagValueTypeInt)),
			},
			{
				ContextInfo: contextInfo,
				Name:        goutil.String("Country"),
				TagDesc:     goutil.String("Country of Users"),
				ValueType:   goutil.Uint32(uint32(entity.TagValueTypeStr)),
			},
		}
		createTagRess = []*CreateTagResponse{
			new(CreateTagResponse),
			new(CreateTagResponse),
		}
	)
	for i, tagReq := range createTagReqs {
		if err := h.tagHandler.CreateTag(ctx, tagReq, createTagRess[i]); err != nil {
			log.Ctx(ctx).Error().Msgf("create tag error: %s", err)
			return err
		}
	}

	var (
		ageTag     = createTagRess[0].Tag
		countryTag = createTagRess[1].Tag
	)

	var (
		createSegmentReq = &CreateSegmentRequest{
			ContextInfo: contextInfo,
			Name:        goutil.String("Millennials or Malaysians"),
			SegmentDesc: goutil.String("Users aged between 18 and 40, or users from Malaysia"),
			Criteria: &entity.Query{
				Queries: []*entity.Query{
					{
						Lookups: []*entity.Lookup{
							{
								TagID: ageTag.ID,
								Op:    entity.LookupOpGt,
								Val:   18,
							},
							{
								TagID: ageTag.ID,
								Op:    entity.LookupOpLt,
								Val:   40,
							},
						},
						Op: entity.QueryOpAnd,
					},
					{
						Lookups: []*entity.Lookup{
							{
								TagID: countryTag.ID,
								Op:    entity.LookupOpEq,
								Val:   "Malaysia",
							},
						},
						Op: entity.QueryOpAnd,
					},
				},
				Op: entity.QueryOpOr,
			},
		}
		createSegmentRes = new(CreateSegmentResponse)
	)
	if err := h.segmentHandler.CreateSegment(ctx, createSegmentReq, createSegmentRes); err != nil {
		log.Ctx(ctx).Error().Msgf("create segment error: %s", err)
		return err
	}

	var (
		now         = uint64(time.Now().Unix())
		resourceIDs = []uint64{ageTag.GetID(), ageTag.GetID(), countryTag.GetID()}
		task        = &entity.Task{
			TenantID:     tenant.ID,
			ResourceID:   nil,
			Status:       entity.TaskStatusSuccess,
			TaskType:     entity.TaskTypeFileUpload,
			ResourceType: entity.ResourceTypeTag,
			ExtInfo: &entity.TaskExtInfo{
				FileID:      goutil.String(""),
				OriFileName: goutil.String("file.csv"),
				Size:        goutil.Uint64(1_000),
				Progress:    goutil.Uint64(100),
			},
			CreatorID:  adminUser.ID,
			CreateTime: goutil.Uint64(now),
			UpdateTime: goutil.Uint64(now),
		}
	)
	for _, resourceID := range resourceIDs {
		task.ResourceID = goutil.Uint64(resourceID)
		_, err := h.taskRepo.Create(ctx, task)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create task error: %s", err)
			return err
		}
	}

	var (
		logInReq = &LogInRequest{
			TenantName: goutil.String(tenantName),
			Username:   goutil.String(username),
			Password:   goutil.String(password),
		}
		logInRes = new(LogInResponse)
	)
	if err := h.userHandler.LogIn(ctx, logInReq, logInRes); err != nil {
		log.Ctx(ctx).Error().Msgf("login user error: %s", err)
		return err
	}

	res.Session = logInRes.Session

	return nil
}
