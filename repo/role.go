package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"strings"
	"time"
)

var (
	ErrRoleNotFound = errutil.NotFoundError(errors.New("role not found"))
)

type Role struct {
	ID         *uint64 `json:"id,omitempty" gorm:"primaryKey"`
	Name       *string `json:"name,omitempty"`
	RoleDesc   *string `json:"role_desc,omitempty"`
	Status     *uint32 `json:"status,omitempty"`
	Actions    *string `json:"actions,omitempty"`
	TenantID   *uint64 `json:"tenant_id,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}

func (m *Role) TableName() string {
	return "role_tab"
}

func (m *Role) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Role) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

func (m *Role) GetActions() string {
	if m != nil && m.Actions != nil {
		return *m.Actions
	}
	return ""
}

type RoleRepo interface {
	GetByID(ctx context.Context, tenantID, roleID uint64) (*entity.Role, error)
	GetManyByIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]*entity.Role, error)
	GetManyByTenantID(ctx context.Context, tenantID uint64) ([]*entity.Role, error)
	Create(ctx context.Context, role *entity.Role) (uint64, error)
	CreateMany(ctx context.Context, roles []*entity.Role) ([]uint64, error)
	Update(ctx context.Context, role *entity.Role) error
	UpdateMany(ctx context.Context, roles []*entity.Role) error
}

type roleRepo struct {
	cacheKeyPrefix string
	baseRepo       BaseRepo
	baseCache      BaseCache
}

func NewRoleRepo(ctx context.Context, baseRepo BaseRepo) (RoleRepo, error) {
	r := &roleRepo{
		cacheKeyPrefix: "role",
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
					log.Ctx(ctx).Error().Msgf("failed to refresh role cache, err: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return r, nil
}

func (r *roleRepo) refreshCache(ctx context.Context) error {
	allRoles, err := r.getMany(ctx, nil, nil, true)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to refresh role cache, err: %v", err)
		return err
	}

	r.baseCache.Flush(ctx)

	log.Ctx(ctx).Info().Msgf("refreshing role cache, %d roles found", len(allRoles))
	for _, role := range allRoles {
		r.setCache(ctx, role)
	}

	return nil
}

func (r *roleRepo) setCache(ctx context.Context, role *entity.Role) {
	r.baseCache.Set(ctx, r.cacheKeyPrefix, role.GetTenantID(), role.GetID(), role)
}

func (r *roleRepo) getFromCache(ctx context.Context, tenantID uint64, uniqKey interface{}) *entity.Role {
	if v, ok := r.baseCache.Get(ctx, r.cacheKeyPrefix, tenantID, uniqKey); ok {
		return v.(*entity.Role)
	}
	return nil
}

func (r *roleRepo) GetByID(ctx context.Context, tenantID, roleID uint64) (*entity.Role, error) {
	role := r.getFromCache(ctx, tenantID, roleID)
	if role != nil {
		return role, nil
	}

	return r.get(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "id",
			Value: roleID,
			Op:    OpEq,
		},
	}, true)
}

func (r *roleRepo) GetManyByIDs(ctx context.Context, tenantID uint64, roleIDs []uint64) ([]*entity.Role, error) {
	var (
		cacheRoles     = make([]*entity.Role, 0, len(roleIDs))
		missingRoleIDs = make([]uint64, 0, len(roleIDs))
	)

	for _, roleID := range roleIDs {
		role := r.getFromCache(ctx, tenantID, roleID)
		if role == nil {
			missingRoleIDs = append(missingRoleIDs, roleID)
		} else {
			cacheRoles = append(cacheRoles, role)
		}
	}

	if len(missingRoleIDs) == 0 {
		return cacheRoles, nil
	}

	dbRoles, err := r.getMany(ctx, goutil.Uint64(tenantID), []*Condition{
		{
			Field: "id",
			Value: missingRoleIDs,
			Op:    OpIn,
		},
	}, true)
	if err != nil {
		return nil, err
	}

	return append(cacheRoles, dbRoles...), nil
}

func (r *roleRepo) Create(ctx context.Context, role *entity.Role) (uint64, error) {
	roleModel := ToRoleModel(role)

	if err := r.baseRepo.Create(ctx, roleModel); err != nil {
		return 0, err
	}

	return roleModel.GetID(), nil
}

func (r *roleRepo) CreateMany(ctx context.Context, roles []*entity.Role) ([]uint64, error) {
	roleModels := make([]*Role, 0, len(roles))
	for _, role := range roles {
		roleModels = append(roleModels, ToRoleModel(role))
	}

	if err := r.baseRepo.CreateMany(ctx, new(Role), roleModels); err != nil {
		return nil, err
	}

	roleIDs := make([]uint64, 0, len(roles))
	for _, roleModel := range roleModels {
		roleIDs = append(roleIDs, roleModel.GetID())
	}

	return roleIDs, nil
}

func (r *roleRepo) Update(ctx context.Context, role *entity.Role) error {
	if err := r.baseRepo.Update(ctx, ToRoleModel(role)); err != nil {
		return err
	}

	r.setCache(ctx, role)
	return nil
}

func (r *roleRepo) UpdateMany(ctx context.Context, roles []*entity.Role) error {
	if err := r.baseRepo.RunTx(ctx, func(ctx context.Context) error {
		for _, role := range roles {
			if err := r.Update(ctx, role); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return err
	}

	for _, role := range roles {
		r.setCache(ctx, role)
	}

	return nil
}

func (r *roleRepo) GetManyByTenantID(ctx context.Context, tenantID uint64) ([]*entity.Role, error) {
	return r.getMany(ctx, goutil.Uint64(tenantID), nil, true)
}

func (r *roleRepo) get(ctx context.Context, tenantID *uint64, conditions []*Condition, filterDelete bool) (*entity.Role, error) {
	role := new(Role)

	if err := r.baseRepo.Get(ctx, role, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	return ToRole(role), nil
}

func (r *roleRepo) getMany(ctx context.Context, tenantID *uint64, conditions []*Condition, filterDelete bool) ([]*entity.Role, error) {
	res, _, err := r.baseRepo.GetMany(ctx, new(Role), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	})
	if err != nil {
		return nil, err
	}

	roles := make([]*entity.Role, len(res))
	for i, m := range res {
		roles[i] = ToRole(m.(*Role))
	}

	return roles, nil
}

func (r *roleRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.RoleStatusDeleted,
			Op:    OpNotEq,
		})

	}
	return conditions
}

func (r *roleRepo) getBaseConditions(tenantID *uint64) []*Condition {
	conditions := make([]*Condition, 0)
	if tenantID != nil {
		conditions = append(conditions, &Condition{
			Field:         "tenant_id",
			Value:         *tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		})
	}
	return conditions
}

func ToRole(role *Role) *entity.Role {
	actions := make([]entity.ActionCode, 0)

	if role.GetActions() != "" {
		parts := strings.Split(role.GetActions(), ",")
		for _, part := range parts {
			actions = append(actions, entity.ActionCode(part))
		}
	}

	return &entity.Role{
		ID:         role.ID,
		Name:       role.Name,
		RoleDesc:   role.RoleDesc,
		Status:     entity.RoleStatus(role.GetStatus()),
		Actions:    actions,
		TenantID:   role.TenantID,
		CreateTime: role.CreateTime,
		UpdateTime: role.UpdateTime,
	}
}

func ToRoleModel(role *entity.Role) *Role {
	actions := make([]string, 0)
	for _, action := range role.Actions {
		actions = append(actions, string(action))
	}

	return &Role{
		ID:         role.ID,
		Name:       role.Name,
		RoleDesc:   role.RoleDesc,
		Status:     goutil.Uint32(uint32(role.GetStatus())),
		Actions:    goutil.String(strings.Join(actions, ",")),
		TenantID:   role.TenantID,
		CreateTime: role.CreateTime,
		UpdateTime: role.UpdateTime,
	}
}
