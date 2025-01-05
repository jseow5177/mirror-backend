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
	ErrTenantNotFound = errutil.NotFoundError(errors.New("tenant not found"))
)

type Tenant struct {
	ID         *uint64 `json:"id,omitempty"`
	Name       *string `json:"name,omitempty"`
	Status     *uint32 `json:"status,omitempty"`
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
	if err := r.baseRepo.Update(ctx, ToTenantModel(tenant)); err != nil {
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
	})
}

func (r *tenantRepo) GetByID(ctx context.Context, tenantID uint64) (*entity.Tenant, error) {
	return r.get(ctx, []*Condition{
		{
			Field: "id",
			Value: tenantID,
			Op:    OpEq,
		},
	})
}

func (r *tenantRepo) get(ctx context.Context, conditions []*Condition) (*entity.Tenant, error) {
	tenant := new(Tenant)

	if err := r.baseRepo.Get(ctx, tenant, &Filter{
		Conditions: r.baseRepo.BuildConditions(r.getBaseConditions(), conditions),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return ToTenant(tenant), nil
}

func (r *tenantRepo) getBaseConditions() []*Condition {
	return []*Condition{
		{
			Field:         "status",
			Value:         entity.TenantStatusDeleted,
			Op:            OpNotEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
}

func (r *tenantRepo) Create(ctx context.Context, tenant *entity.Tenant) (uint64, error) {
	tenantModel := ToTenantModel(tenant)

	if err := r.baseRepo.Create(ctx, tenantModel); err != nil {
		return 0, err
	}

	return tenantModel.GetID(), nil
}

func ToTenant(tenant *Tenant) *entity.Tenant {
	return &entity.Tenant{
		ID:         tenant.ID,
		Name:       tenant.Name,
		Status:     entity.TenantStatus(tenant.GetStatus()),
		CreateTime: tenant.CreateTime,
		UpdateTime: tenant.UpdateTime,
	}
}

func ToTenantModel(tenant *entity.Tenant) *Tenant {
	return &Tenant{
		ID:         tenant.ID,
		Name:       tenant.Name,
		Status:     goutil.Uint32(uint32(tenant.GetStatus())),
		CreateTime: tenant.CreateTime,
		UpdateTime: tenant.UpdateTime,
	}
}
