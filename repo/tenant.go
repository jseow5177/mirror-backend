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
	Get(ctx context.Context, f *Filter) (*entity.Tenant, error)
	Create(ctx context.Context, tenant *entity.Tenant) (uint64, error)
	Update(ctx context.Context, tenant *entity.Tenant) error
	Close(ctx context.Context) error
}

type tenantRepo struct {
	orm *gorm.DB
}

func NewTenantRepo(_ context.Context, mysqlCfg config.MySQL) (TenantRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &tenantRepo{orm: orm}, nil
}

func (r *tenantRepo) Update(_ context.Context, tenant *entity.Tenant) error {
	if err := r.orm.Updates(ToTenantModel(tenant)).Error; err != nil {
		return err
	}

	return nil
}

func (r *tenantRepo) Get(_ context.Context, f *Filter) (*entity.Tenant, error) {
	sql, args := ToSqlWithArgs(f)
	tenant := new(Tenant)

	if err := r.orm.Where(sql, args...).First(tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, err
	}

	return ToTenant(tenant), nil
}

func (r *tenantRepo) Create(_ context.Context, tenant *entity.Tenant) (uint64, error) {
	tenantModel := ToTenantModel(tenant)

	if err := r.orm.Create(tenantModel).Error; err != nil {
		return 0, err
	}

	return tenantModel.GetID(), nil
}

func (r *tenantRepo) Close(_ context.Context) error {
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
