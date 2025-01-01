package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"strings"
	"time"
)

type UserHandler interface {
	CreateUser(ctx context.Context, req *CreateUserRequest, res *CreateUserResponse) error
	CreateUsers(ctx context.Context, req *CreateUsersRequest, res *CreateUsersResponse) error
}

type userHandler struct {
	userRepo repo.UserRepo
}

func NewUserHandler(userRepo repo.UserRepo) UserHandler {
	return &userHandler{
		userRepo: userRepo,
	}
}

type CreateUserRequest struct {
	TenantID    *uint64 `json:"tenant_id,omitempty"`
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty"`
}

func GetCreateUserValidator(optionalPassword bool) validator.Validator {
	return validator.MustForm(map[string]validator.Validator{
		"tenant_id":    &validator.UInt64{},
		"email":        EmailValidator(false),
		"display_name": DisplayNameValidator(true),
		"password":     PasswordValidator(optionalPassword),
	})
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
	if err := GetCreateUserValidator(false).Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	user, err := req.ToUser()
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

type CreateUsersRequest struct {
	Users []*CreateUserRequest `json:"users,omitempty"`
}

type CreateUsersResponse struct {
	Users []*entity.User `json:"users,omitempty"`
}

var CreateUsersValidator = validator.MustForm(map[string]validator.Validator{
	"users": &validator.Slice{
		Validator: GetCreateUserValidator(false),
	},
})

func (h *userHandler) CreateUsers(ctx context.Context, req *CreateUsersRequest, res *CreateUsersResponse) error {
	if err := CreateUsersValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	users := make([]*entity.User, 0, len(req.Users))
	for i, r := range req.Users {
		u, err := r.ToUser()
		if err != nil {
			log.Ctx(ctx).Error().Msgf("failed to convert to user: %v, i: %v", err, i)
			return err
		}
		users = append(users, u)
	}

	userIDs, err := h.userRepo.BatchCreate(ctx, users)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("failed to create users: %v", err)
		return err
	}

	for i, id := range userIDs {
		users[i].ID = goutil.Uint64(id)
	}

	res.Users = users

	return nil
}
