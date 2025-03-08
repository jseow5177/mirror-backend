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
	"strings"
)

type UserHandler interface {
	CreateUsers(ctx context.Context, req *CreateUsersRequest, res *CreateUsersResponse) error
	GetUsers(ctx context.Context, req *GetUsersRequest, res *GetUsersResponse) error
	InitUser(ctx context.Context, req *InitUserRequest, res *InitUserResponse) error
	LogIn(ctx context.Context, req *LogInRequest, res *LogInResponse) error
	LogOut(ctx context.Context, req *LogOutRequest, _ *LogOutResponse) error
	Me(ctx context.Context, req *GetUserRequest, res *GetUserResponse) error
}

type userHandler struct {
	cfg            *config.Config
	txService      repo.TxService
	emailService   dep.EmailService
	userRepo       repo.UserRepo
	tenantRepo     repo.TenantRepo
	activationRepo repo.ActivationRepo
	sessionRepo    repo.SessionRepo
	roleRepo       repo.RoleRepo
	userRoleRepo   repo.UserRoleRepo
}

func NewUserHandler(cfg *config.Config, txService repo.TxService, emailService dep.EmailService,
	userRepo repo.UserRepo, tenantRepo repo.TenantRepo, activationRepo repo.ActivationRepo, sessionRepo repo.SessionRepo,
	roleRepo repo.RoleRepo, userRoleRepo repo.UserRoleRepo) UserHandler {
	return &userHandler{
		cfg:            cfg,
		txService:      txService,
		emailService:   emailService,
		userRepo:       userRepo,
		tenantRepo:     tenantRepo,
		activationRepo: activationRepo,
		sessionRepo:    sessionRepo,
		roleRepo:       roleRepo,
		userRoleRepo:   userRoleRepo,
	}
}

type LogOutRequest struct {
	ContextInfo
}

type LogOutResponse struct{}

var LogOutValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
})

func (h *userHandler) LogOut(ctx context.Context, req *LogOutRequest, _ *LogOutResponse) error {
	if err := LogOutValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if err := h.sessionRepo.DeleteByUserID(ctx, req.GetUserID()); err != nil {
		log.Ctx(ctx).Error().Msgf("delete session err: %v", err)
		return err
	}

	return nil
}

type LogInRequest struct {
	TenantID   *uint64 `json:"tenant_id,omitempty"`
	TenantName *string `json:"tenant_name,omitempty"`
	Username   *string `json:"username,omitempty"`
	Password   *string `json:"password,omitempty"`
}

func (r *LogInRequest) GetTenantID() uint64 {
	if r != nil && r.TenantID != nil {
		return *r.TenantID
	}
	return 0
}

func (r *LogInRequest) GetTenantName() string {
	if r != nil && r.TenantName != nil {
		return *r.TenantName
	}
	return ""
}

func (r *LogInRequest) GetUsername() string {
	if r != nil && r.Username != nil {
		return *r.Username
	}
	return ""
}

func (r *LogInRequest) GetPassword() string {
	if r != nil && r.Password != nil {
		return *r.Password
	}
	return ""
}

type LogInResponse struct {
	Session *entity.Session `json:"session,omitempty"`
}

var LogInValidator = validator.MustForm(map[string]validator.Validator{
	"tenant_id": &validator.UInt64{
		Optional: true,
	},
	"tenant_name": &validator.String{
		Optional: true,
	},
	"username": &validator.String{},
	"password": &validator.String{},
})

func (h *userHandler) LogIn(ctx context.Context, req *LogInRequest, res *LogInResponse) error {
	if err := LogInValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	stdErr := errutil.ValidationError(errors.New("incorrect credentials"))

	var (
		err    error
		tenant *entity.Tenant
	)
	if req.TenantID != nil {
		tenant, err = h.tenantRepo.GetByID(ctx, req.GetTenantID())
	} else {
		tenant, err = h.tenantRepo.GetByName(ctx, req.GetTenantName())
	}
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get tenant error: %v", err)
		return stdErr
	}

	user, err := h.userRepo.GetByUsername(ctx, tenant.GetID(), req.GetUsername())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get user error: %v", err)
		return stdErr
	}

	if !user.ComparePassword(req.GetPassword()) {
		return stdErr
	}

	sess, err := entity.NewSession(user.GetID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("new session error: %v", err)
		return err
	}

	id, err := h.sessionRepo.Create(ctx, sess)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create session error: %v", err)
		return err
	}

	sess.ID = goutil.Uint64(id)
	res.Session = sess

	return nil
}

type InitUserRequest struct {
	Token    *string `json:"token,omitempty"`
	Password *string `json:"password,omitempty"`
}

func (r *InitUserRequest) GetToken() string {
	if r != nil && r.Token != nil {
		return *r.Token
	}
	return ""
}

func (r *InitUserRequest) GetPassword() string {
	if r != nil && r.Password != nil {
		return *r.Password
	}
	return ""
}

type InitUserResponse struct {
	User    *entity.User    `json:"user,omitempty"`
	Session *entity.Session `json:"session,omitempty"`
}

var InitUserValidator = validator.MustForm(map[string]validator.Validator{
	"token":    &validator.String{},
	"password": PasswordValidator(false),
})

func (h *userHandler) InitUser(ctx context.Context, req *InitUserRequest, res *InitUserResponse) error {
	if err := InitUserValidator.Validate(req); err != nil {
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

	user, err := h.userRepo.GetByID(ctx, act.GetTargetID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get user failed: %v", err)
		return err
	}

	if !user.IsPending() {
		log.Ctx(ctx).Info().Msgf("user is not pending, token: %v", req.GetToken())
		return errutil.ValidationError(errors.New("invalid token/user"))
	}

	if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
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

		var (
			logInReq = &LogInRequest{
				TenantID: user.TenantID,
				Username: user.Username,
				Password: req.Password,
			}
			logInRes = new(LogInResponse)
		)
		if err := h.LogIn(ctx, logInReq, logInRes); err != nil {
			log.Ctx(ctx).Error().Msgf("login user failed: %v", err)
			return err
		}

		res.User = user
		res.Session = logInRes.Session

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type CreateUserRequest struct {
	Email       *string `json:"email,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Password    *string `json:"password,omitempty"`
	RoleID      *uint64 `json:"role_id,omitempty"`
}

var CreateUserValidator = validator.MustForm(map[string]validator.Validator{
	"email":        EmailValidator(false),
	"display_name": DisplayNameValidator(true),
	"password":     PasswordValidator(true),
	"role_id":      &validator.UInt64{},
})

func (r *CreateUserRequest) GetRoleID() uint64 {
	if r != nil && r.RoleID != nil {
		return *r.RoleID
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

func (r *CreateUserRequest) ToUser(tenantID uint64) (*entity.User, error) {
	return entity.NewUser(tenantID, r.GetEmail(), r.GetPassword(), r.GetDisplayName())
}

func (r *CreateUserRequest) extractUsernameFromEmail() (string, error) {
	parts := strings.Split(r.GetEmail(), "@")
	if len(parts) != 2 || parts[0] == "" {
		return "", fmt.Errorf("invalid email: %v", r.GetEmail())
	}
	return parts[0], nil
}

type CreateUsersRequest struct {
	ContextInfo
	Users []*CreateUserRequest `json:"users,omitempty"`
}

type CreateUsersResponse struct {
	Users []*entity.User `json:"users,omitempty"`
}

func (r *CreateUsersRequest) ToUsers() ([]*entity.User, error) {
	users := make([]*entity.User, 0, len(r.Users))

	for _, req := range r.Users {
		user, err := req.ToUser(r.GetTenantID())
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

var CreateUsersValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, true),
	"users": &validator.Slice{
		Optional:  false,
		MinLen:    1,
		MaxLen:    5,
		Validator: CreateUserValidator,
	},
})

func (h *userHandler) CreateUsers(ctx context.Context, req *CreateUsersRequest, res *CreateUsersResponse) error {
	if err := CreateUsersValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	users, err := req.ToUsers()
	if err != nil {
		log.Ctx(ctx).Error().Msgf("convert to users error: %v", err)
		return err
	}

	usersMap := make(map[string]bool, len(users))
	for _, u := range users {
		if usersMap[u.GetEmail()] {
			return errutil.ValidationError(fmt.Errorf("duplicate user email found: %v", u.GetEmail()))
		} else {
			usersMap[u.GetEmail()] = true
		}
	}

	emails := make([]string, 0)
	for _, u := range users {
		emails = append(emails, u.GetEmail())
	}
	existUsers, err := h.userRepo.GetManyByEmails(ctx, req.GetTenantID(), emails)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get users error: %v", err)
		return err
	}

	if len(existUsers) > 0 {
		existEmails := make([]string, 0)
		for _, u := range existUsers {
			existEmails = append(existEmails, u.GetEmail())
		}
		return errutil.ValidationError(fmt.Errorf("emails already exist: %v", existEmails))
	}

	var (
		pendingUsers = make([]*entity.User, 0)
		acts         = make([]*entity.Activation, 0)
		userRoles    = make([]*entity.UserRole, 0)
	)
	if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
		// create users
		userIDs, err := h.userRepo.CreateMany(ctx, users)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create users failed: %v", err)
			return err
		}

		for i, u := range users {
			// create new activation tokens if user is pending
			if u.IsPending() {
				act, err := entity.NewActivation(userIDs[i], entity.TokenTypeUser)
				if err != nil {
					log.Ctx(ctx).Error().Msgf("new activation failed: %v", err)
					return err
				}
				acts = append(acts, act)
				pendingUsers = append(pendingUsers, u)
			}

			u.ID = goutil.Uint64(userIDs[i])

			// create roles for each user
			var (
				roleID   = req.Users[i].GetRoleID()
				roles    = make(map[uint64]*entity.Role) // cache for roles
				userRole = &entity.UserRole{
					TenantID:   u.TenantID,
					UserID:     u.ID,
					RoleID:     goutil.Uint64(roleID),
					CreateTime: u.CreateTime,
					UpdateTime: u.UpdateTime,
				}
			)

			if _, ok := roles[roleID]; !ok {
				role, err := h.roleRepo.GetByID(ctx, req.GetTenantID(), req.Users[i].GetRoleID())
				if err != nil {
					log.Ctx(ctx).Error().Msgf("get role failed: %v", err)
					return err
				}
				roles[roleID] = role
			}
			userRoles = append(userRoles, userRole)

			u.Role = roles[roleID]
		}

		if len(userRoles) != 0 {
			_, err = h.userRoleRepo.CreateMany(ctx, userRoles)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("create roles failed: %v", err)
				return err
			}
		}

		if len(acts) != 0 {
			if err = h.activationRepo.CreateMany(ctx, acts); err != nil {
				log.Ctx(ctx).Error().Msgf("create activations failed: %v", err)
				return err
			}
		}

		return nil
	}); err != nil {
		return err
	}

	go func(ctx context.Context) {
		for i, user := range pendingUsers {
			act := acts[i]

			initUserLink, err := goutil.BuildURL(h.cfg.WebPage.Domain, h.cfg.WebPage.Paths.InitUser, map[string]string{
				"token": act.GetToken(),
			})
			if err != nil {
				log.Ctx(ctx).Error().Msgf("create init user link failed: %v", err)
				continue
			}

			emailVars := map[string]string{
				"username":       user.GetUsername(),
				"tenant_name":    req.GetTenantName(),
				"role":           user.Role.GetName(),
				"init_user_link": initUserLink,
			}

			var content bytes.Buffer
			if err := initUserTmpl.Execute(&content, emailVars); err != nil {
				log.Ctx(ctx).Error().Msgf("build email template failed: %v, tenant: %v, user: %v", err,
					req.GetTenantName(), user.GetUsername())
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
					req.GetTenantName(), user.GetUsername())
				continue
			}
		}
	}(context.WithoutCancel(ctx))

	res.Users = users

	return nil
}

type GetUserRequest struct {
	ContextInfo
}

type GetUserResponse struct {
	User *entity.User `json:"user,omitempty"`
}

var GetUserValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
})

func (h *userHandler) Me(_ context.Context, req *GetUserRequest, res *GetUserResponse) error {
	if err := GetUserValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	res.User = req.User
	return nil
}

type GetUsersRequest struct {
	ContextInfo

	Keyword    *string          `json:"keyword,omitempty"`
	Status     []uint32         `json:"status,omitempty"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

func (req *GetUsersRequest) GetKeyword() string {
	if req != nil && req.Keyword != nil {
		return *req.Keyword
	}
	return ""
}

type GetUsersResponse struct {
	Users      []*entity.User   `json:"users"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

var GetUsersValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
	"keyword": &validator.String{
		Optional: true,
	},
	"status": &validator.Slice{
		Optional:  true,
		Validator: new(userStatusValidator),
	},
	"pagination": PaginationValidator(),
})

func (h *userHandler) GetUsers(ctx context.Context, req *GetUsersRequest, res *GetUsersResponse) error {
	if err := GetUsersValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.Pagination == nil {
		req.Pagination = new(repo.Pagination)
	}

	status := make([]entity.UserStatus, len(req.Status))
	for i, s := range req.Status {
		status[i] = entity.UserStatus(s)
	}

	users, pagination, err := h.userRepo.GetManyByKeyword(
		ctx, req.GetTenantID(), req.GetKeyword(), status, req.Pagination)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get users failed: %v", err)
		return err
	}

	roles, err := h.roleRepo.GetManyByTenantID(ctx, req.GetTenantID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get roles failed: %v", err)
		return err
	}

	rolesMap := make(map[uint64]*entity.Role)
	for _, role := range roles {
		rolesMap[role.GetID()] = role
	}

	userIDs := make([]uint64, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.GetID())
	}

	userRoles, err := h.userRoleRepo.GetManyByUserIDs(ctx, req.GetTenantID(), userIDs)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get user roles failed: %v", err)
		return err
	}

	if len(userIDs) != len(userRoles) {
		log.Ctx(ctx).Error().Msgf("not enough user roles, userIDs: %v", userIDs)
		return err
	}

	userIDsToRoles := make(map[uint64]*entity.Role)
	for _, userRole := range userRoles {
		userIDsToRoles[userRole.GetUserID()] = rolesMap[userRole.GetRoleID()]
	}

	for _, u := range users {
		u.Role = userIDsToRoles[u.GetID()]
	}

	res.Users = users
	res.Pagination = pagination

	return nil
}
