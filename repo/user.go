package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"gorm.io/driver/mysql"
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
	Get(ctx context.Context, f *Filter) (*entity.User, error)
	Create(ctx context.Context, user *entity.User) (uint64, error)
	BatchCreate(_ context.Context, users []*entity.User) ([]uint64, error)
	Close(ctx context.Context) error
}

type userRepo struct {
	orm *gorm.DB
}

func NewUserRepo(_ context.Context, mysqlCfg config.MySQL) (UserRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &userRepo{orm: orm}, nil
}

func (r *userRepo) Get(_ context.Context, f *Filter) (*entity.User, error) {
	sql, args := ToSqlWithArgs(f)
	user := new(User)

	if err := r.orm.Where(sql, args...).First(user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	return ToUser(user), nil
}

func (r *userRepo) Create(_ context.Context, user *entity.User) (uint64, error) {
	userModel := ToUserModel(user)

	if err := r.orm.Create(userModel).Error; err != nil {
		return 0, err
	}

	return userModel.GetID(), nil
}

func (r *userRepo) BatchCreate(_ context.Context, users []*entity.User) ([]uint64, error) {
	userModels := make([]*User, 0)
	for _, user := range users {
		userModels = append(userModels, ToUserModel(user))
	}

	if err := r.orm.Create(userModels).Error; err != nil {
		return nil, err
	}

	userIDs := make([]uint64, 0, len(userModels))
	for _, userModel := range userModels {
		userIDs = append(userIDs, userModel.GetID())
	}

	return userIDs, nil
}

func (r *userRepo) Close(_ context.Context) error {
	if r.orm != nil {
		sqlDB, err := r.orm.DB()
		if err != nil {
			return err
		}

		err = sqlDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
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
