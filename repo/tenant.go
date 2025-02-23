package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
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
	baseRepo BaseRepo
}

func NewTenantRepo(_ context.Context, baseRepo BaseRepo) TenantRepo {
	return &tenantRepo{baseRepo: baseRepo}
}

func (r *tenantRepo) Update(ctx context.Context, tenant *entity.Tenant) error {
	tenantModel, err := ToTenantModel(tenant)
	if err != nil {
		return err
	}

	if err := r.baseRepo.Update(ctx, tenantModel); err != nil {
		return err
	}

	return nil
}

func (r *tenantRepo) GetByName(ctx context.Context, tenantName string) (*entity.Tenant, error) {
	return r.get(ctx, []*Condition{
		{
			Field: "name",
			Value: tenantName,
			Op:    OpEq,
		},
	}, true)
}

func (r *tenantRepo) GetByID(ctx context.Context, tenantID uint64) (*entity.Tenant, error) {
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
