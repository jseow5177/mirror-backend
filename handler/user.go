package handler

import (
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

type UserHandler interface {
	CreateUser(ctx context.Context, req *CreateUserRequest, res *CreateUserResponse) error
}

type userHandler struct {
	userRepo   repo.UserRepo
	tenantRepo repo.TenantRepo
}

func NewUserHandler(userRepo repo.UserRepo, tenantRepo repo.TenantRepo) UserHandler {
	return &userHandler{
		userRepo:   userRepo,
		tenantRepo: tenantRepo,
	}
}

type IsUserPendingInitRequest struct {
	UserID *uint64 `json:"user_id,omitempty"`
}

type IsUserPendingInitResponse struct {
	IsPendingInit *bool `json:"is_pending_init,omitempty"`
}

var IsUserPendingInitValidator = validator.MustForm(map[string]validator.Validator{
	"user_id": &validator.UInt64{},
})

func (h *userHandler) IsUserPendingInit(ctx context.Context, req *IsUserPendingInitRequest, res *IsUserPendingInitResponse) error {
	if err := IsUserPendingInitValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	return nil
}

type InitUserRequest struct {
	Password *string `json:"password,omitempty"`
}

type InitUserResponse struct {
	User *entity.User `json:"user,omitempty"`
}

type CreateUserRequest struct {
	TenantID    *uint64 `json:"tenant_id,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty"`
}

type CreateUserOptionalFields struct {
	TenantID bool
	Password bool
}

func GetCreateUserValidator(opt CreateUserOptionalFields) validator.Validator {
	return validator.MustForm(map[string]validator.Validator{
		"tenant_id": &validator.UInt64{
			Optional: opt.TenantID,
		},
		"email":        EmailValidator(false),
		"display_name": DisplayNameValidator(true),
		"password":     PasswordValidator(opt.Password),
	})
}

func (r *CreateUserRequest) GetTenantID() uint64 {
	if r != nil && r.TenantID != nil {
		return *r.TenantID
	}
	return 0
}

func (r *CreateUserRequest) GetEmail() string {
	if r != nil && r.Email != nil {
		return *r.Email
	}
	return ""
}

func (r *CreateUserRequest) GetDisplayName() string {
	if r != nil && r.DisplayName != nil {
		return *r.DisplayName
	}
	return ""
}

func (r *CreateUserRequest) GetPassword() string {
	if r != nil && r.Password != nil {
		return *r.Password
	}
	return ""
}

func (r *CreateUserRequest) ToUser() (*entity.User, error) {
	now := uint64(time.Now().Unix())

	username, err := r.extractUsernameFromEmail()
	if err != nil {
		return nil, err
	}

	var (
		password string
		status   = entity.UserStatusPending
	)
	if r.GetPassword() != "" {
		password, err = goutil.BCrypt(r.GetPassword())
		if err != nil {
			return nil, err
		}
		status = entity.UserStatusNormal
	}

	user := &entity.User{
		TenantID:    r.TenantID,
		Email:       r.Email,
		Username:    goutil.String(username),
		Password:    goutil.String(password),
		DisplayName: goutil.String(r.GetDisplayName()),
		Status:      status,
		CreateTime:  goutil.Uint64(now),
		UpdateTime:  goutil.Uint64(now),
	}

	return user, nil
}

func (r *CreateUserRequest) extractUsernameFromEmail() (string, error) {
	parts := strings.Split(r.GetEmail(), "@")
	if len(parts) != 2 || parts[0] == "" {
		return "", fmt.Errorf("invalid email: %v", r.GetEmail())
	}
	return parts[0], nil
}

type CreateUserResponse struct {
	User *entity.User `json:"user,omitempty"`
}

func (h *userHandler) CreateUser(ctx context.Context, req *CreateUserRequest, res *CreateUserResponse) error {
	if err := GetCreateUserValidator(CreateUserOptionalFields{}).Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	tenant, err := h.tenantRepo.GetByID(ctx, req.GetTenantID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tenant error: %v", err)
		return err
	}

	if !tenant.IsNormal() {
		return errutil.ValidationError(errors.New("tenant is not status normal"))
	}

	user, err := h.userRepo.GetByEmail(ctx, req.GetTenantID(), req.GetEmail())
	if err == nil {
		return errutil.ConflictError(errors.New("user already exists"))
	}

	if !errors.Is(err, repo.ErrUserNotFound) {
		log.Ctx(ctx).Error().Msgf("get user error: %v", err)
		return err
	}

	user, err = req.ToUser()
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to convert to user: %v", err)
		return err
	}

	id, err := h.userRepo.Create(ctx, user)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to create user: %v", err)
		return err
	}

	user.ID = goutil.Uint64(id)
	res.User = user

	return nil
}
