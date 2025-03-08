package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

var (
	ErrEmailNotFound = errutil.NotFoundError(errors.New("email not found"))
)

type Email struct {
	ID         *uint64
	Name       *string
	EmailDesc  *string
	Json       *string
	Html       *string
	Status     *uint32
	CreatorID  *uint64
	TenantID   *uint64
	CreateTime *uint64
	UpdateTime *uint64
}

func (m *Email) TableName() string {
	return "email_tab"
}

func (m *Email) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Email) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

type EmailRepo interface {
	Create(ctx context.Context, email *entity.Email) (uint64, error)
	GetByID(ctx context.Context, tenantID, emailID uint64) (*entity.Email, error)
	GetManyByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Email, *Pagination, error)
}

type emailRepo struct {
	baseRepo BaseRepo
}

func NewEmailRepo(_ context.Context, baseRepo BaseRepo) EmailRepo {
	return &emailRepo{baseRepo: baseRepo}
}

func (r *emailRepo) Create(ctx context.Context, email *entity.Email) (uint64, error) {
	emailModel := ToEmailModel(email)

	if err := r.baseRepo.Create(ctx, emailModel); err != nil {
		return 0, err
	}

	return emailModel.GetID(), nil
}

func (r *emailRepo) GetByID(ctx context.Context, tenantID, emailID uint64) (*entity.Email, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "id",
			Value: emailID,
			Op:    OpEq,
		},
	}, true)
}

func (r *emailRepo) GetManyByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Email, *Pagination, error) {
	return r.getMany(ctx, tenantID, []*Condition{
		{
			Field:         "LOWER(name)",
			Value:         fmt.Sprintf("%%%s%%", keyword),
			Op:            OpLike,
			NextLogicalOp: LogicalOpOr,
			OpenBracket:   true,
		},
		{
			Field:        "LOWER(email_desc)",
			Value:        fmt.Sprintf("%%%s%%", keyword),
			Op:           OpLike,
			CloseBracket: true,
		},
	}, true, p)
}

func (r *emailRepo) getMany(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool, p *Pagination) ([]*entity.Email, *Pagination, error) {
	res, pNew, err := r.baseRepo.GetMany(ctx, new(Email), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
		Pagination: p,
	})
	if err != nil {
		return nil, nil, err
	}

	emails := make([]*entity.Email, 0, len(res))
	for _, m := range res {
		emails = append(emails, ToEmail(m.(*Email)))
	}

	return emails, pNew, nil
}

func (r *emailRepo) get(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (*entity.Email, error) {
	email := new(Email)

	if err := r.baseRepo.Get(ctx, email, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrEmailNotFound
		}
		return nil, err
	}

	return ToEmail(email), nil
}

func (r *emailRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.EmailStatusDeleted,
			Op:    OpNotEq,
		})

	}
	return conditions
}

func (r *emailRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
}

func ToEmailModel(email *entity.Email) *Email {
	return &Email{
		ID:         email.ID,
		Name:       email.Name,
		EmailDesc:  email.EmailDesc,
		Json:       email.Json,
		Html:       email.Html,
		Status:     goutil.Uint32(uint32(email.GetStatus())),
		CreatorID:  email.CreatorID,
		TenantID:   email.TenantID,
		CreateTime: email.CreateTime,
		UpdateTime: email.UpdateTime,
	}
}

func ToEmail(email *Email) *entity.Email {
	return &entity.Email{
		ID:         email.ID,
		Name:       email.Name,
		EmailDesc:  email.EmailDesc,
		Json:       email.Json,
		Html:       email.Html,
		Status:     entity.EmailStatus(email.GetStatus()),
		CreatorID:  email.CreatorID,
		TenantID:   email.TenantID,
		CreateTime: email.CreateTime,
		UpdateTime: email.UpdateTime,
	}
}
