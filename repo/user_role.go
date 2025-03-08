package repo

import (
	"cdp/entity"
	"context"
)

type UserRole struct {
	ID         *uint64 `json:"id,omitempty"`
	TenantID   *uint64 `json:"tenant_id,omitempty"`
	UserID     *uint64 `json:"user_id,omitempty"`
	RoleID     *uint64 `json:"role_id,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}

func (m *UserRole) TableName() string {
	return "user_role_tab"
}

func (m *UserRole) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

type UserRoleRepo interface {
	Create(ctx context.Context, userRole *entity.UserRole) (uint64, error)
	CreateMany(ctx context.Context, userRoles []*entity.UserRole) ([]uint64, error)
	Update(ctx context.Context, userRole *entity.UserRole) error
	GetManyByUserIDs(ctx context.Context, tenantID uint64, userIDs []uint64) ([]*entity.UserRole, error)
}

type userRoleRepo struct {
	baseRepo BaseRepo
}

func NewUserRoleRepo(_ context.Context, baseRepo BaseRepo) UserRoleRepo {
	return &userRoleRepo{
		baseRepo: baseRepo,
	}
}

func (r *userRoleRepo) Create(ctx context.Context, userRole *entity.UserRole) (uint64, error) {
	userRoleModel := ToUserRoleModel(userRole)

	if err := r.baseRepo.Create(ctx, userRoleModel); err != nil {
		return 0, err
	}

	return userRoleModel.GetID(), nil
}

func (r *userRoleRepo) CreateMany(ctx context.Context, userRoles []*entity.UserRole) ([]uint64, error) {
	userRoleModels := make([]*UserRole, 0, len(userRoles))
	for _, userRole := range userRoles {
		userRoleModels = append(userRoleModels, ToUserRoleModel(userRole))
	}

	if err := r.baseRepo.CreateMany(ctx, new(UserRole), userRoleModels); err != nil {
		return nil, err
	}

	userRoleIds := make([]uint64, 0, len(userRoles))
	for _, userRoleModel := range userRoleModels {
		userRoleIds = append(userRoleIds, userRoleModel.GetID())
	}

	return userRoleIds, nil
}

func (r *userRoleRepo) Update(ctx context.Context, userRole *entity.UserRole) error {
	return r.baseRepo.Update(ctx, ToUserRoleModel(userRole))
}

func (r *userRoleRepo) GetManyByUserIDs(ctx context.Context, tenantID uint64, userIDs []uint64) ([]*entity.UserRole, error) {
	return r.getMany(ctx, tenantID, []*Condition{
		{
			Field: "user_id",
			Value: userIDs,
			Op:    OpIn,
		},
	})
}

func (r *userRoleRepo) getMany(ctx context.Context, tenantID uint64, conditions []*Condition) ([]*entity.UserRole, error) {
	res, _, err := r.baseRepo.GetMany(ctx, new(UserRole), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), conditions...),
	})
	if err != nil {
		return nil, err
	}

	userRoles := make([]*entity.UserRole, len(res))
	for i, m := range res {
		userRoles[i] = ToUserRole(m.(*UserRole))
	}

	return userRoles, nil
}

func (r *userRoleRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
}

func ToUserRole(userRole *UserRole) *entity.UserRole {
	return &entity.UserRole{
		ID:         userRole.ID,
		TenantID:   userRole.TenantID,
		UserID:     userRole.UserID,
		RoleID:     userRole.RoleID,
		CreateTime: userRole.CreateTime,
		UpdateTime: userRole.UpdateTime,
	}
}

func ToUserRoleModel(userRole *entity.UserRole) *UserRole {
	return &UserRole{
		ID:         userRole.ID,
		TenantID:   userRole.TenantID,
		UserID:     userRole.UserID,
		RoleID:     userRole.RoleID,
		CreateTime: userRole.CreateTime,
		UpdateTime: userRole.UpdateTime,
	}
}
