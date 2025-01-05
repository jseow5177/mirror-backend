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
)

type UserHandler interface {
	CreateUser(ctx context.Context, req *CreateUserRequest, res *CreateUserResponse) error
	IsUserPendingInit(ctx context.Context, req *IsUserPendingInitRequest, res *IsUserPendingInitResponse) error
	InitUser(ctx context.Context, req *InitUserRequest, res *InitUserResponse) error
}

type userHandler struct {
	userRepo       repo.UserRepo
	tenantRepo     repo.TenantRepo
	activationRepo repo.ActivationRepo
}

func NewUserHandler(userRepo repo.UserRepo, tenantRepo repo.TenantRepo, activationRepo repo.ActivationRepo) UserHandler {
	return &userHandler{
		userRepo:       userRepo,
		tenantRepo:     tenantRepo,
		activationRepo: activationRepo,
	}
}

type IsUserPendingInitRequest struct {
	Token *string `json:"token,omitempty"`
}

func (r *IsUserPendingInitRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

type IsUserPendingInitResponse struct {
	IsPending *bool        `json:"is_pending,omitempty"`
	User      *entity.User `json:"user,omitempty"`
}

func (r *IsUserPendingInitResponse) GetIsPending() bool {
	if r != nil && r.IsPending != nil {
		return *r.IsPending
	}
	return false
}

var IsUserPendingInitValidator = validator.MustForm(map[string]validator.Validator{
	"token": &validator.String{},
})

func (h *userHandler) IsUserPendingInit(ctx context.Context, req *IsUserPendingInitRequest, res *IsUserPendingInitResponse) error {
	if err := IsUserPendingInitValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	token, err := goutil.Base64Decode(req.GetToken())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("decode base64 token failed: %v", err)
		return err
	}

	act, err := h.activationRepo.GetByTokenHash(ctx, goutil.Sha256(token), entity.TokenTypeUser)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get activation token failed: %v", err)
		return err
	}

	userID := act.GetTargetID()

	user, err := h.userRepo.GetByID(ctx, userID)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get user failed: %v", err)
		return err
	}

	res.IsPending = goutil.Bool(user.IsPending())
	res.User = user

	return nil
}

type InitUserRequest struct {
	Token    *string `json:"token,omitempty"`
	Password *string `json:"password,omitempty"`
}

func (r *InitUserRequest) GetPassword() string {
	if r != nil && r.Password != nil {
		return *r.Password
	}
	return ""
}

type InitUserResponse struct {
	User *entity.User `json:"user,omitempty"`
}

var InitUserValidator = validator.MustForm(map[string]validator.Validator{
	"token":    &validator.String{},
	"password": &validator.String{},
})

func (h *userHandler) InitUser(ctx context.Context, req *InitUserRequest, res *InitUserResponse) error {
	if err := InitUserValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	isUserPendingInitRequest := &IsUserPendingInitRequest{
		Token: req.Token,
	}
	isUserPendingInitResponse := new(IsUserPendingInitResponse)

	if err := h.IsUserPendingInit(ctx, isUserPendingInitRequest, isUserPendingInitResponse); err != nil {
		return err
	}

	user := isUserPendingInitResponse.User
	if !isUserPendingInitResponse.GetIsPending() {
		return errutil.ValidationError(fmt.Errorf("user is not pending, name: %v, tenant_id: %v", user.GetUsername(), user.GetTenantID()))
	}

	newUser, err := entity.NewUser(
		user.GetTenantID(), user.GetEmail(), req.GetPassword(), user.GetDisplayName(),
	)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create new user failed: %v", err)
		return err
	}

	user.Update(newUser)

	if err := h.userRepo.Update(ctx, user); err != nil {
		log.Ctx(ctx).Error().Msgf("update user failed: %v", err)
		return err
	}

	res.User = user

	return nil
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
	return entity.NewUser(r.GetTenantID(), r.GetEmail(), r.GetPassword(), r.GetDisplayName())
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
	if err := GetCreateUserValidator(CreateUserOptionalFields{
		Password: true,
	}).Validate(req); err != nil {
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
