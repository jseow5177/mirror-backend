package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"time"
)

var (
	ErrUserNotFound = errutil.NotFoundError(errors.New("user not found"))
)

type User struct {
	ID          *uint64 `json:"id,omitempty"`
	TenantID    *uint64 `json:"tenant_id,omitempty"`
	Email       *string `json:"email,omitempty"`
	Username    *string `json:"username,omitempty"`
	Password    *string `json:"password,omitempty"`
	DisplayName *string `json:"display_name,omitempty"`
	Status      *uint32 `json:"status,omitempty"`
	CreateTime  *uint64 `json:"create_time,omitempty"`
	UpdateTime  *uint64 `json:"update_time,omitempty"`
}

func (m *User) TableName() string {
	return "user_tab"
}

func (m *User) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *User) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

type UserRepo interface {
	Create(ctx context.Context, user *entity.User) (uint64, error)
	CreateMany(ctx context.Context, users []*entity.User) ([]uint64, error)
	Update(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, userID uint64) (*entity.User, error)
	GetByEmail(ctx context.Context, tenantID uint64, email string) (*entity.User, error)
	GetByUsername(ctx context.Context, tenantID uint64, username string) (*entity.User, error)
	GetManyByKeyword(ctx context.Context, tenantID uint64,
		keyword string, status []entity.UserStatus, p *Pagination) ([]*entity.User, *Pagination, error)
	GetManyByEmails(ctx context.Context, tenantID uint64, email []string) ([]*entity.User, error)
}

type userRepo struct {
	cacheKeyPrefix string
	baseRepo       BaseRepo
	baseCache      BaseCache
}

func NewUserRepo(ctx context.Context, baseRepo BaseRepo, baseCache BaseCache) (UserRepo, error) {
	r := &userRepo{
		cacheKeyPrefix: "user",
		baseRepo:       baseRepo,
		baseCache:      baseCache,
	}

	if err := r.refreshCache(ctx); err != nil {
		return nil, err
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := r.refreshCache(ctx); err != nil {
					log.Ctx(ctx).Error().Msgf("failed to refresh user cache, err: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return r, nil
}

func (r *userRepo) refreshCache(ctx context.Context) error {
	allUsers, _, err := r.getMany(ctx, nil, nil, true, nil)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("refreshing user cache, %d user found", len(allUsers))
	for _, user := range allUsers {
		r.setCache(ctx, user)
	}

	return nil
}

func (r *userRepo) setCache(ctx context.Context, user *entity.User) {
	r.baseCache.Set(ctx, r.cacheKeyPrefix, 0, user.GetID(), user)
	r.baseCache.Set(ctx, r.cacheKeyPrefix, user.GetTenantID(), user.GetEmail(), user)
	r.baseCache.Set(ctx, r.cacheKeyPrefix, user.GetTenantID(), user.GetUsername(), user)
}

func (r *userRepo) getFromCache(ctx context.Context, tenantID uint64, uniqKey interface{}) *entity.User {
	if v, ok := r.baseCache.Get(ctx, r.cacheKeyPrefix, tenantID, uniqKey); ok {
		return v.(*entity.User)
	}
	return nil
}

func (r *userRepo) GetManyByKeyword(ctx context.Context, tenantID uint64,
	keyword string, status []entity.UserStatus, p *Pagination) ([]*entity.User, *Pagination, error) {
	conditions := []*Condition{
		{
			OpenBracket:   true,
			Field:         "LOWER(email)",
			Value:         fmt.Sprintf("%%%s%%", keyword),
			Op:            OpLike,
			NextLogicalOp: LogicalOpOr,
		},
		{
			Field:         "LOWER(username)",
			Value:         fmt.Sprintf("%%%s%%", keyword),
			Op:            OpLike,
			CloseBracket:  true,
			NextLogicalOp: LogicalOpAnd,
		},
	}
	if len(status) > 0 {
		conditions = append(conditions, &Condition{
			Field: "status",
			Value: status,
			Op:    OpIn,
		})
	}
	return r.getMany(ctx, goutil.Uint64(tenantID), conditions, true, p)
}

func (r *userRepo) GetManyByEmails(ctx context.Context, tenantID uint64, emails []string) ([]*entity.User, error) {
	var (
		cacheUsers    = make([]*entity.User, 0, len(emails))
		missingEmails = make([]string, 0, len(emails))
	)

	for _, email := range emails {
		user := r.getFromCache(ctx, tenantID, email)
		if user == nil {
			missingEmails = append(missingEmails, email)
		} else {
			cacheUsers = append(cacheUsers, user)
		}
	}

	if len(missingEmails) == 0 {
		return cacheUsers, nil
	}

	dbUsers, _, err := r.getMany(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "email",
			Value: missingEmails,
			Op:    OpIn,
		},
	}, true, nil)
	if err != nil {
		return nil, err
	}

	return append(cacheUsers, dbUsers...), err
}

func (r *userRepo) GetByID(ctx context.Context, userID uint64) (*entity.User, error) {
	user := r.getFromCache(ctx, 0, userID)
	if user != nil {
		return user, nil
	}

	return r.get(ctx, nil, []*Condition{
		{
			Field: "id",
			Value: userID,
			Op:    OpEq,
		},
	}, true)
}

func (r *userRepo) GetByEmail(ctx context.Context, tenantID uint64, email string) (*entity.User, error) {
	user := r.getFromCache(ctx, tenantID, email)
	if user != nil {
		return user, nil
	}

	return r.get(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "email",
			Value: email,
			Op:    OpEq,
		},
	}, true)
}

func (r *userRepo) GetByUsername(ctx context.Context, tenantID uint64, username string) (*entity.User, error) {
	user := r.getFromCache(ctx, tenantID, username)
	if user != nil {
		return user, nil
	}

	return r.get(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "username",
			Value: username,
			Op:    OpEq,
		},
	}, true)
}

func (r *userRepo) getMany(ctx context.Context, tenantID *uint64, conditions []*Condition, filterDelete bool, p *Pagination) ([]*entity.User, *Pagination, error) {
	res, pNew, err := r.baseRepo.GetMany(ctx, new(User), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
		Pagination: p,
	})
	if err != nil {
		return nil, nil, err
	}

	roles := make([]*entity.User, len(res))
	for i, m := range res {
		roles[i] = ToUser(m.(*User))
	}

	return roles, pNew, nil
}

func (r *userRepo) get(ctx context.Context, tenantID *uint64, conditions []*Condition, filterDelete bool) (*entity.User, error) {
	user := new(User)

	if err := r.baseRepo.Get(ctx, user, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return ToUser(user), nil
}

func (r *userRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.UserStatusDeleted,
			Op:    OpNotEq,
		})
	}
	return conditions
}

func (r *userRepo) getBaseConditions(tenantID *uint64) []*Condition {
	conditions := make([]*Condition, 0)

	if tenantID != nil {
		conditions = append(conditions, &Condition{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		})
	}

	return conditions
}

func (r *userRepo) Create(ctx context.Context, user *entity.User) (uint64, error) {
	userModel := ToUserModel(user)

	if err := r.baseRepo.Create(ctx, userModel); err != nil {
		return 0, err
	}

	return userModel.GetID(), nil
}

func (r *userRepo) Update(ctx context.Context, user *entity.User) error {
	if err := r.baseRepo.Update(ctx, ToUserModel(user)); err != nil {
		return err
	}

	r.setCache(ctx, user)

	return nil
}

func (r *userRepo) CreateMany(ctx context.Context, users []*entity.User) ([]uint64, error) {
	userModels := make([]*User, 0, len(users))
	for _, user := range users {
		userModels = append(userModels, ToUserModel(user))
	}

	if err := r.baseRepo.CreateMany(ctx, new(User), userModels); err != nil {
		return nil, err
	}

	userIDs := make([]uint64, 0, len(userModels))
	for _, userModel := range userModels {
		userIDs = append(userIDs, userModel.GetID())
	}

	return userIDs, nil
}

func ToUser(user *User) *entity.User {
	return &entity.User{
		ID:          user.ID,
		TenantID:    user.TenantID,
		Email:       user.Email,
		Username:    user.Username,
		Password:    user.Password,
		DisplayName: user.DisplayName,
		Status:      entity.UserStatus(user.GetStatus()),
		CreateTime:  user.CreateTime,
		UpdateTime:  user.UpdateTime,
	}
}

func ToUserModel(user *entity.User) *User {
	return &User{
		ID:          user.ID,
		TenantID:    user.TenantID,
		Email:       user.Email,
		Username:    user.Username,
		DisplayName: user.DisplayName,
		Password:    user.Password,
		Status:      goutil.Uint32(uint32(user.GetStatus())),
		CreateTime:  user.CreateTime,
		UpdateTime:  user.UpdateTime,
	}
}
