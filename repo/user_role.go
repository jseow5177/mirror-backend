package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"time"
)

var (
	ErrUserRoleNotFound = errutil.NotFoundError(errors.New("user role not found"))
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
	GetByUserID(ctx context.Context, tenantID, userID uint64) (*entity.UserRole, error)
	GetManyByUserIDs(ctx context.Context, tenantID uint64, userIDs []uint64) ([]*entity.UserRole, error)
}

type userRoleRepo struct {
	cacheKeyPrefix string
	baseRepo       BaseRepo
	baseCache      BaseCache
}

func NewUserRoleRepo(ctx context.Context, baseRepo BaseRepo) (UserRoleRepo, error) {
	r := &userRoleRepo{
		cacheKeyPrefix: "user_role",
		baseRepo:       baseRepo,
		baseCache:      NewBaseCache(ctx),
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
					log.Ctx(ctx).Error().Msgf("failed to refresh user role cache, err: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return r, nil
}

func (r *userRoleRepo) refreshCache(ctx context.Context) error {
	allUserRoles, err := r.getMany(ctx, nil, nil)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to refresh user role cache, err: %v", err)
		return err
	}

	r.baseCache.Flush(ctx)

	log.Ctx(ctx).Info().Msgf("refreshing user roles cache, %d user roles found", len(allUserRoles))
	for _, userRole := range allUserRoles {
		r.setCache(ctx, userRole)
	}

	return nil
}

func (r *userRoleRepo) setCache(ctx context.Context, userRole *entity.UserRole) {
	r.baseCache.Set(ctx, r.cacheKeyPrefix, userRole.GetTenantID(), userRole.GetUserID(), userRole)
}

func (r *userRoleRepo) getFromCache(ctx context.Context, tenantID uint64, uniqKey interface{}) *entity.UserRole {
	if v, ok := r.baseCache.Get(ctx, r.cacheKeyPrefix, tenantID, uniqKey); ok {
		return v.(*entity.UserRole)
	}
	return nil
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
	if err := r.baseRepo.Update(ctx, ToUserRoleModel(userRole)); err != nil {
		return err
	}

	r.setCache(ctx, userRole)

	return nil
}

func (r *userRoleRepo) GetManyByUserIDs(ctx context.Context, tenantID uint64, userIDs []uint64) ([]*entity.UserRole, error) {
	var (
		cacheUserRoles = make([]*entity.UserRole, 0, len(userIDs))
		missingUserIDs = make([]uint64, 0, len(userIDs))
	)

	for _, userID := range userIDs {
		userRole := r.getFromCache(ctx, tenantID, userID)
		if userRole == nil {
			missingUserIDs = append(missingUserIDs, userID)
		} else {
			cacheUserRoles = append(cacheUserRoles, userRole)
		}
	}

	if len(missingUserIDs) == 0 {
		return cacheUserRoles, nil
	}

	dbRole, err := r.getMany(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "user_id",
			Value: missingUserIDs,
			Op:    OpIn,
		},
	})
	if err != nil {
		return nil, err
	}

	return append(cacheUserRoles, dbRole...), nil
}

func (r *userRoleRepo) GetByUserID(ctx context.Context, tenantID, userID uint64) (*entity.UserRole, error) {
	userRole := r.getFromCache(ctx, tenantID, userID)
	if userRole != nil {
		return userRole, nil
	}

	return r.get(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "user_id",
			Value: userID,
			Op:    OpEq,
		},
	})
}

func (r *userRoleRepo) get(ctx context.Context, tenantID *uint64, conditions []*Condition) (*entity.UserRole, error) {
	userRole := new(UserRole)

	if err := r.baseRepo.Get(ctx, userRole, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), conditions...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserRoleNotFound
		}
		return nil, err
	}

	return ToUserRole(userRole), nil
}

func (r *userRoleRepo) getMany(ctx context.Context, tenantID *uint64, conditions []*Condition) ([]*entity.UserRole, error) {
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

func (r *userRoleRepo) getBaseConditions(tenantID *uint64) []*Condition {
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
