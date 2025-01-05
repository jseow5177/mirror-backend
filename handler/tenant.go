package handler

import (
	"bytes"
	"cdp/config"
	"cdp/dep"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"path"
	"time"
)

type TenantHandler interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error
	GetTenant(ctx context.Context, req *GetTenantRequest, res *GetTenantResponse) error
	InitTenant(ctx context.Context, req *InitTenantRequest, res *InitTenantResponse) error
	IsTenantPendingActivation(ctx context.Context, req *IsTenantPendingActivationRequest, res *IsTenantPendingActivationResponse) error
}

type tenantHandler struct {
	cfg            *config.Config
	txService      repo.TxService
	tenantRepo     repo.TenantRepo
	userRepo       repo.UserRepo
	activationRepo repo.ActivationRepo
	emailService   dep.EmailService
}

func NewTenantHandler(
	cfg *config.Config,
	txService repo.TxService,
	tenantRepo repo.TenantRepo,
	userRepo repo.UserRepo,
	activationRepo repo.ActivationRepo,
	emailService dep.EmailService,
) TenantHandler {
	return &tenantHandler{
		cfg:            cfg,
		txService:      txService,
		tenantRepo:     tenantRepo,
		userRepo:       userRepo,
		activationRepo: activationRepo,
		emailService:   emailService,
	}
}

type CreateTenantRequest struct {
	Name *string `json:"name,omitempty"`
}

func (r *CreateTenantRequest) GetName() string {
	if r != nil && r.Name != nil {
		return *r.Name
	}
	return ""
}

func (r *CreateTenantRequest) ToTenant() *entity.Tenant {
	now := time.Now()
	return &entity.Tenant{
		Name:       r.Name,
		Status:     entity.TenantStatusPending,
		CreateTime: goutil.Uint64(uint64(now.Unix())),
		UpdateTime: goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateTenantResponse struct {
	Tenant *entity.Tenant `json:"tenant,omitempty"`
	Token  *string        `json:"token,omitempty"`
}

var CreateTenantValidator = validator.MustForm(map[string]validator.Validator{
	"name": ResourceNameValidator(false),
})

func (h *tenantHandler) CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error {
	if err := CreateTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	_, err := h.tenantRepo.GetByName(ctx, req.GetName())
	if err == nil {
		return errutil.ConflictError(errors.New("tenant already exists"))
	}

	if !errors.Is(err, repo.ErrTenantNotFound) {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	tenant := req.ToTenant()

	if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
		id, err := h.tenantRepo.Create(ctx, tenant)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create tenant failed: %v", err)
			return err
		}

		act, err := entity.NewActivation(id, entity.TokenTypeTenant)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create activation token failed: %v", err)
			return err
		}

		if _, err := h.activationRepo.Create(ctx, act); err != nil {
			log.Ctx(ctx).Error().Msgf("create activation token failed: %v", err)
			return err
		}

		tenant.ID = goutil.Uint64(id)
		res.Tenant = tenant
		res.Token = goutil.String(act.ToEncodedToken())

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type GetTenantRequest struct {
	Name *string `json:"name,omitempty"`
}

func (r *GetTenantRequest) GetName() string {
	if r != nil && r.Name != nil {
		return *r.Name
	}
	return ""
}

type GetTenantResponse struct {
	Tenant *entity.Tenant `json:"tenant,omitempty"`
}

var GetTenantValidator = validator.MustForm(map[string]validator.Validator{
	"name": ResourceNameValidator(false),
})

func (h *tenantHandler) GetTenant(ctx context.Context, req *GetTenantRequest, res *GetTenantResponse) error {
	if err := GetTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	tenant, err := h.tenantRepo.GetByName(ctx, req.GetName())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	res.Tenant = tenant

	return nil
}

type IsTenantPendingActivationRequest struct {
	Token *string `json:"token,omitempty"`
}

func (r *IsTenantPendingActivationRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type IsTenantPendingActivationResponse struct {
	IsPending *bool          `json:"is_pending,omitempty"`
	Tenant    *entity.Tenant `json:"tenant,omitempty"`
}

func (r *IsTenantPendingActivationResponse) GetIsPending() bool {
	if r != nil && r.IsPending != nil {
		return *r.IsPending
	}
	return false
}

var IsTenantPendingActivationValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
})

func (h *tenantHandler) IsTenantPendingActivation(ctx context.Context, req *IsTenantPendingActivationRequest, res *IsTenantPendingActivationResponse) error {
	if err := IsTenantPendingActivationValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	token, err := goutil.Base64Decode(req.GetToken())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("decode base64 token failed: %v", err)
		return err
	}

	act, err := h.activationRepo.GetByTokenHash(ctx, goutil.Sha256(token), entity.TokenTypeTenant)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get activation token failed: %v", err)
		return err
	}

	tenantID := act.GetTargetID()

	tenant, err := h.tenantRepo.GetByID(ctx, tenantID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	res.IsPending = goutil.Bool(tenant.IsPending())
	res.Tenant = tenant

	return nil
}

type InitTenantRequest struct {
	Token      *string              `json:"token,omitempty"`
	User       *CreateUserRequest   `json:"user,omitempty"`
	OtherUsers []*CreateUserRequest `json:"other_users,omitempty"`
}

func (r *InitTenantRequest) ToUsers(tenantID uint64) ([]*entity.User, error) {
	users := make([]*entity.User, 0, len(r.OtherUsers))

	firstUser, err := r.User.ToUser()
	if err != nil {
		return nil, err
	}
	firstUser.TenantID = goutil.Uint64(tenantID)
	users = append(users, firstUser)

	for _, u := range r.OtherUsers {
		otherUser, err := u.ToUser()
		if err != nil {
			return nil, err
		}
		otherUser.TenantID = goutil.Uint64(tenantID)
		users = append(users, otherUser)
	}

	return users, nil
}

func (r *InitTenantRequest) GetUser() *CreateUserRequest {
	if r != nil && r.User != nil {
		return r.User
	}
	return nil
}

func (r *InitTenantRequest) GetOtherUsers() []*CreateUserRequest {
	if r != nil && r.OtherUsers != nil {
		return r.OtherUsers
	}
	return nil
}

func (r *InitTenantRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type InitTenantResponse struct {
	Tenant *entity.Tenant `json:"tenant,omitempty"`
}

var InitTenantValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
	"user": GetCreateUserValidator(CreateUserOptionalFields{
		TenantID: true,
	}),
	"other_users": &validator.Slice{
		Validator: GetCreateUserValidator(CreateUserOptionalFields{
			TenantID: true,
			Password: true,
		}),
	},
})

func (h *tenantHandler) InitTenant(ctx context.Context, req *InitTenantRequest, res *InitTenantResponse) error {
	if err := InitTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	isTenantPendingActivationReq := &IsTenantPendingActivationRequest{
		Token: req.Token,
	}
	isTenantPendingActivationRes := new(IsTenantPendingActivationResponse)

	if err := h.IsTenantPendingActivation(ctx, isTenantPendingActivationReq, isTenantPendingActivationRes); err != nil {
		return err
	}

	tenant := isTenantPendingActivationRes.Tenant
	if !isTenantPendingActivationRes.GetIsPending() {
		return errutil.ValidationError(fmt.Errorf("tenant is not pending, name: %v", tenant.GetName()))
	}

	users, err := req.ToUsers(tenant.GetID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("to users failed: %v", err)
		return err
	}

	var userIDs []uint64
	if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
		if userIDs, err = h.userRepo.CreateMany(ctx, users); err != nil {
			log.Ctx(ctx).Error().Msgf("create users failed: %v", err)
			return err
		}

		tenant.Update(&entity.Tenant{
			Status: entity.TenantStatusNormal,
		})

		if err := h.tenantRepo.Update(ctx, tenant); err != nil {
			log.Ctx(ctx).Error().Msgf("update tenant failed: %v", err)
			return err
		}

		res.Tenant = tenant

		return nil
	}); err != nil {
		return err
	}

	go func(ctx context.Context) {
		for i, user := range users {
			act, err := entity.NewActivation(userIDs[i], entity.TokenTypeUser)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("new activation failed: %v", err)
				continue
			}

			initUserPage := path.Join(
				h.cfg.WebPage.Domain,
				fmt.Sprintf(h.cfg.WebPage.Paths.InitUser, act.ToEncodedToken()))

			emailVars := map[string]string{
				"username":     user.GetUsername(),
				"tenant_name":  tenant.GetName(),
				"welcome_page": initUserPage,
			}

			var content bytes.Buffer
			if err := initUserTmpl.Execute(&content, emailVars); err != nil {
				log.Ctx(ctx).Error().Msgf("build email template failed: %v, tenant: %v, user: %v", err,
					tenant.GetName(), user.GetUsername())
				continue
			}

			if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
				if _, err := h.activationRepo.Create(ctx, act); err != nil {
					log.Ctx(ctx).Error().Msgf("create activation failed: %v", err)
					return err
				}

				sendEmailReq := &dep.SendSmtpEmail{
					From: &dep.Sender{
						Email: h.cfg.InternalSender,
					},
					To: []*dep.Receiver{
						{Email: user.GetEmail()},
					},
					Subject:     "Welcome to Mirror!",
					HtmlContent: string(content.Bytes()),
				}
				if err := h.emailService.SendEmail(ctx, sendEmailReq); err != nil {
					log.Ctx(ctx).Error().Msgf("send email failed: %v, tenant: %v, user: %v", err,
						tenant.GetName(), user.GetUsername())
					return err
				}

				return nil
			}); err != nil {
				continue
			}
		}
	}(context.WithoutCancel(ctx))

	return nil
}
