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
	"time"
)

type TenantHandler interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error
	GetTenant(ctx context.Context, req *GetTenantRequest, res *GetTenantResponse) error
	InitTenant(ctx context.Context, req *InitTenantRequest, _ *InitTenantResponse) error
}

type tenantHandler struct {
	cfg          *config.Config
	tenantRepo   repo.TenantRepo
	userRepo     repo.UserRepo
	emailService dep.EmailService
}

func NewTenantHandler(cfg *config.Config, tenantRepo repo.TenantRepo,
	userRepo repo.UserRepo, emailService dep.EmailService) TenantHandler {
	return &tenantHandler{
		cfg:          cfg,
		tenantRepo:   tenantRepo,
		userRepo:     userRepo,
		emailService: emailService,
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
}

var CreateTenantValidator = validator.MustForm(map[string]validator.Validator{
	"name": ResourceNameValidator(false),
})

func (h *tenantHandler) CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error {
	if err := CreateTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	_, err := h.getValidTenantByName(ctx, req.GetName())
	if err == nil {
		return errutil.ConflictError(errors.New("tenant already exists"))
	}

	if !errors.Is(err, repo.ErrTenantNotFound) {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	tenant := req.ToTenant()

	id, err := h.tenantRepo.Create(ctx, tenant)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create tenant failed: %v", err)
		return err
	}

	tenant.ID = goutil.Uint64(id)
	res.Tenant = tenant

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

	tenant, err := h.getValidTenantByName(ctx, req.GetName())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	res.Tenant = tenant

	return nil
}

type InitTenantRequest struct {
	TenantID   *uint64              `json:"tenant_id,omitempty"`
	User       *CreateUserRequest   `json:"user,omitempty"`
	OtherUsers []*CreateUserRequest `json:"other_users,omitempty"`
}

func (r *InitTenantRequest) ToUsers() ([]*entity.User, error) {
	users := make([]*entity.User, 0, len(r.OtherUsers))

	firstUser, err := r.User.ToUser()
	if err != nil {
		return nil, err
	}
	users = append(users, firstUser)

	for _, u := range r.OtherUsers {
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

func (r *InitTenantRequest) GetTenantID() uint64 {
	if r != nil && r.TenantID != nil {
		return *r.TenantID
	}
	return 0
}

type InitTenantResponse struct{}

var InitTenantValidator = validator.MustForm(map[string]validator.Validator{
	"tenant_id": &validator.UInt64{},
	"user":      GetCreateUserValidator(false),
	"other_users": &validator.Slice{
		Validator: GetCreateUserValidator(true),
	},
})

func (h *tenantHandler) InitTenant(ctx context.Context, req *InitTenantRequest, _ *InitTenantResponse) error {
	req.User.TenantID = req.TenantID
	for _, u := range req.OtherUsers {
		u.TenantID = req.TenantID
	}

	if err := InitTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	tenant, err := h.getValidTenantByID(ctx, req.GetTenantID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	if !tenant.IsPending() {
		return errutil.ValidationError(fmt.Errorf("tenant is not pending, name: %v", tenant.GetName()))
	}

	users, err := req.ToUsers()
	if err != nil {
		log.Ctx(ctx).Error().Msgf("to users failed: %v", err)
		return err
	}

	if _, err := h.userRepo.BatchCreate(ctx, users); err != nil {
		log.Ctx(ctx).Error().Msgf("create users failed: %v", err)
		return err
	}

	hasChange := tenant.Update(&entity.Tenant{
		Status: entity.TenantStatusNormal,
	})

	if hasChange {
		if err := h.tenantRepo.Update(ctx, tenant); err != nil {
			log.Ctx(ctx).Error().Msgf("update tenant failed: %v", err)
			return err
		}
	}

	go func() {
		for _, user := range users {
			emailVars := map[string]string{
				"username":     user.GetUsername(),
				"tenant_name":  tenant.GetName(),
				"welcome_page": h.cfg.WebPages.WelcomePage,
			}

			var content bytes.Buffer
			if err := welcomeEmailTmpl.Execute(&content, emailVars); err != nil {
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
	}()

	return nil
}

func (h *tenantHandler) getValidTenantByName(ctx context.Context, name string) (*entity.Tenant, error) {
	f := &repo.Filter{
		Conditions: []*repo.Condition{
			{
				Field:         "name",
				Value:         name,
				Op:            repo.OpEq,
				NextLogicalOp: repo.LogicalOpAnd,
			},
			{
				Field: "status",
				Value: entity.TenantStatusDeleted,
				Op:    repo.OpNotEq,
			},
		},
	}

	return h.tenantRepo.Get(ctx, f)
}

func (h *tenantHandler) getValidTenantByID(ctx context.Context, tenantID uint64) (*entity.Tenant, error) {
	f := &repo.Filter{
		Conditions: []*repo.Condition{
			{
				Field:         "id",
				Value:         tenantID,
				Op:            repo.OpEq,
				NextLogicalOp: repo.LogicalOpAnd,
			},
			{
				Field: "status",
				Value: entity.TenantStatusDeleted,
				Op:    repo.OpNotEq,
			},
		},
	}

	return h.tenantRepo.Get(ctx, f)
}
