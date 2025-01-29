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
)

type TenantHandler interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error
	GetTenant(ctx context.Context, req *GetTenantRequest, res *GetTenantResponse) error
	InitTenant(ctx context.Context, req *InitTenantRequest, res *InitTenantResponse) error
	IsTenantPendingInit(ctx context.Context, req *IsTenantPendingInitRequest, res *IsTenantPendingInitResponse) error
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
	return entity.NewTenant(r.GetName(), entity.TenantStatusPending)
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
		res.Token = act.Token

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

type IsTenantPendingInitRequest struct {
	Token *string `json:"token,omitempty"`
}

func (r *IsTenantPendingInitRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type IsTenantPendingInitResponse struct {
	IsPending *bool          `json:"is_pending,omitempty"`
	Tenant    *entity.Tenant `json:"tenant,omitempty"`
}

func (r *IsTenantPendingInitResponse) GetIsPending() bool {
	if r != nil && r.IsPending != nil {
		return *r.IsPending
	}
	return false
}

var IsTenantPendingInitValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
})

func (h *tenantHandler) IsTenantPendingInit(ctx context.Context, req *IsTenantPendingInitRequest, res *IsTenantPendingInitResponse) error {
	if err := IsTenantPendingInitValidator.Validate(req); err != nil {
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

	r.User.TenantID = goutil.Uint64(tenantID)
	firstUser, err := r.User.ToUser()
	if err != nil {
		return nil, err
	}
	users = append(users, firstUser)

	for _, u := range r.OtherUsers {
		u.TenantID = goutil.Uint64(tenantID)
		otherUser, err := u.ToUser()
		if err != nil {
			return nil, err
		}
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
	Tenant          *entity.Tenant       `json:"tenant,omitempty"`
	UserActivations []*entity.Activation `json:"user_activations,omitempty"`
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

	isTenantPendingActivationReq := &IsTenantPendingInitRequest{
		Token: req.Token,
	}
	isTenantPendingActivationRes := new(IsTenantPendingInitResponse)

	if err := h.IsTenantPendingInit(ctx, isTenantPendingActivationReq, isTenantPendingActivationRes); err != nil {
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

	var (
		acts         = make([]*entity.Activation, 0)
		pendingUsers = make([]*entity.User, 0)
	)
	if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
		userIDs, err := h.userRepo.CreateMany(ctx, users)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create users failed: %v", err)
			return err
		}

		for i, u := range users {
			if u.IsPending() {
				act, err := entity.NewActivation(userIDs[i], entity.TokenTypeUser)
				if err != nil {
					log.Ctx(ctx).Error().Msgf("new activation failed: %v", err)
					return err
				}
				acts = append(acts, act)
				pendingUsers = append(pendingUsers, u)
			}
		}

		_, err = h.activationRepo.CreateMany(ctx, acts)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create activations failed: %v", err)
			return err
		}

		tenant.Update(
			entity.NewTenant(
				tenant.GetName(),
				entity.TenantStatusNormal,
			),
		)

		if err := h.tenantRepo.Update(ctx, tenant); err != nil {
			log.Ctx(ctx).Error().Msgf("update tenant failed: %v", err)
			return err
		}

		res.Tenant = tenant
		res.UserActivations = acts

		return nil
	}); err != nil {
		return err
	}

	go func(ctx context.Context) {
		for i, user := range pendingUsers {
			act := acts[i]

			initUserPage := path.Join(
				h.cfg.WebPage.Domain,
				fmt.Sprintf(h.cfg.WebPage.Paths.InitUser, act.GetToken()),
			)

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
				continue
			}
		}
	}(context.WithoutCancel(ctx))

	return nil
}
