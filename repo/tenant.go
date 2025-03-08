package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
	"time"
)

var (
	ErrTenantNotFound = errutil.NotFoundError(errors.New("tenant not found"))
)

type Tenant struct {
	ID         *uint64 `json:"id,omitempty"`
	Name       *string `json:"name,omitempty"`
	Status     *uint32 `json:"status,omitempty"`
	ExtInfo    *string `json:"ext_info,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}

func (m *Tenant) TableName() string {
	return "tenant_tab"
}

func (m *Tenant) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Tenant) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

func (m *Tenant) GetExtInfo() string {
	if m != nil && m.ExtInfo != nil {
		return *m.ExtInfo
	}
	return ""
}

type TenantRepo interface {
	Create(ctx context.Context, tenant *entity.Tenant) (uint64, error)
	Update(ctx context.Context, tenant *entity.Tenant) error
	GetByName(ctx context.Context, tenantName string) (*entity.Tenant, error)
	GetByID(ctx context.Context, tenantID uint64) (*entity.Tenant, error)
}

type tenantRepo struct {
	cacheKeyPrefix string
	baseRepo       BaseRepo
	baseCache      BaseCache
}

func NewTenantRepo(ctx context.Context, baseRepo BaseRepo, baseCache BaseCache) (TenantRepo, error) {
	r := &tenantRepo{
		cacheKeyPrefix: "tenant",
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
					log.Ctx(ctx).Error().Msgf("failed to refresh tenant cache, err: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return r, nil
}

func (r *tenantRepo) refreshCache(ctx context.Context) error {
	allTenants, err := r.getMany(ctx, nil, true)
	if err != nil {
		return err
	}

	log.Ctx(ctx).Info().Msgf("refreshing tenant cache, %d tenants found", len(allTenants))
	for _, tenant := range allTenants {
		r.setCache(ctx, tenant)
	}

	return nil
}

func (r *tenantRepo) setCache(ctx context.Context, tenant *entity.Tenant) {
	r.baseCache.Set(ctx, r.cacheKeyPrefix, 0, tenant.GetID(), tenant)
	r.baseCache.Set(ctx, r.cacheKeyPrefix, 0, tenant.GetName(), tenant)
}

func (r *tenantRepo) getFromCache(ctx context.Context, uniqKey interface{}) *entity.Tenant {
	if v, ok := r.baseCache.Get(ctx, r.cacheKeyPrefix, 0, uniqKey); ok {
		return v.(*entity.Tenant)
	}
	return nil
}

func (r *tenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	tenantModel, err := ToTenantModel(tenant)
	if err != nil {
		return err
	}

	if err := r.baseRepo.Update(ctx, tenantModel); err != nil {
		return err
	}

	r.setCache(ctx, tenant)

	return nil
}

func (r *tenantRepo) GetByName(ctx context.Context, tenantName string) (*entity.Tenant, error) {
	tenant := r.getFromCache(ctx, tenantName)
	if tenant != nil {
		return tenant, nil
	}

	return r.get(ctx, []*Condition{
		{
			Field: "name",
			Value: tenantName,
			Op:    OpEq,
		},
	}, true)
}

func (r *tenantRepo) GetByID(ctx context.Context, tenantID uint64) (*entity.Tenant, error) {
	tenant := r.getFromCache(ctx, tenantID)
	if tenant != nil {
		return tenant, nil
	}

	return r.get(ctx, []*Condition{
		{
			Field: "id",
			Value: tenantID,
			Op:    OpEq,
		},
	}, true)
}

func (r *tenantRepo) get(ctx context.Context, conditions []*Condition, filterDelete bool) (*entity.Tenant, error) {
	tenant := new(Tenant)

	if err := r.baseRepo.Get(ctx, tenant, &Filter{
		Conditions: append(r.getBaseConditions(), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return ToTenant(tenant)
}

func (r *tenantRepo) getMany(ctx context.Context, conditions []*Condition, filterDelete bool) ([]*entity.Tenant, error) {
	res, _, err := r.baseRepo.GetMany(ctx, new(Tenant), &Filter{
		Conditions: append(r.getBaseConditions(), r.mayAddDeleteFilter(conditions, filterDelete)...),
	})
	if err != nil {
		return nil, err
	}

	tenants := make([]*entity.Tenant, len(res))
	for i, m := range res {
		tenant, err := ToTenant(m.(*Tenant))
		if err != nil {
			return nil, err
		}
		tenants[i] = tenant
	}

	return tenants, nil
}

func (r *tenantRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.TenantStatusDeleted,
			Op:    OpNotEq,
		})
	}
	return conditions
}

func (r *tenantRepo) getBaseConditions() []*Condition {
	return []*Condition{}
}

func (r *tenantRepo) Create(ctx context.Context, tenant *entity.Tenant) (uint64, error) {
	tenantModel, err := ToTenantModel(tenant)
	if err != nil {
		return 0, err
	}

	if err := r.baseRepo.Create(ctx, tenantModel); err != nil {
		return 0, err
	}

	return tenantModel.GetID(), nil
}

func ToTenant(tenant *Tenant) (*entity.Tenant, error) {
	extInfo := new(entity.TenantExtInfo)
	if err := json.Unmarshal([]byte(tenant.GetExtInfo()), extInfo); err != nil {
		return nil, err
	}

	return &entity.Tenant{
		ID:         tenant.ID,
		Name:       tenant.Name,
		Status:     entity.TenantStatus(tenant.GetStatus()),
		ExtInfo:    extInfo,
		CreateTime: tenant.CreateTime,
		UpdateTime: tenant.UpdateTime,
	}, nil
}

func ToTenantModel(tenant *entity.Tenant) (*Tenant, error) {
	extInfo, err := tenant.GetExtInfo().ToString()
	if err != nil {
		return nil, err
	}

	return &Tenant{
		ID:         tenant.ID,
		Name:       tenant.Name,
		Status:     goutil.Uint32(uint32(tenant.GetStatus())),
		ExtInfo:    goutil.String(extInfo),
		CreateTime: tenant.CreateTime,
		UpdateTime: tenant.UpdateTime,
	}, nil
}
