package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"gorm.io/gorm"
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
}

type userRepo struct {
	baseRepo BaseRepo
}

func NewUserRepo(_ context.Context, baseRepo BaseRepo) UserRepo {
	return &userRepo{baseRepo: baseRepo}
}

func (r *userRepo) GetByID(ctx context.Context, userID uint64) (*entity.User, error) {
	return r.get(ctx, 0, []*Condition{
		{
			Field: "id",
			Value: userID,
			Op:    OpEq,
		},
	}, true)
}

func (r *userRepo) GetByEmail(ctx context.Context, tenantID uint64, email string) (*entity.User, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "email",
			Value: email,
			Op:    OpEq,
		},
	}, true)
}

func (r *userRepo) GetByUsername(ctx context.Context, tenantID uint64, username string) (*entity.User, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "username",
			Value: username,
			Op:    OpEq,
		},
	}, true)
}

func (r *userRepo) get(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (*entity.User, error) {
	user := new(User)

	baseConditions := make([]*Condition, 0)
	if tenantID != 0 {
		baseConditions = append(baseConditions, r.getBaseConditions(tenantID)...)
	}

	if err := r.baseRepo.Get(ctx, user, &Filter{
		Conditions: append(baseConditions, r.mayAddDeleteFilter(conditions, filterDelete)...),
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

func (r *userRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
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
