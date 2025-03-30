package repo

import (
	"cdp/entity"
	"context"
	"errors"
	"gorm.io/gorm"
)

var (
	ErrSenderNotFound = errors.New("sender not found")
)

type Sender struct {
	ID         *uint64 `json:"id,omitempty"`
	TenantID   *uint64 `json:"tenant_id,omitempty"`
	Name       *string `json:"name,omitempty"`
	LocalPart  *string `json:"local_part,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}

func (m *Sender) TableName() string {
	return "sender_tab"
}

func (m *Sender) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

type SenderRepo interface {
	Create(ctx context.Context, sender *entity.Sender) (uint64, error)
	GetManyByTenantID(ctx context.Context, tenantID uint64) ([]*entity.Sender, error)
	GetByNameAndLocalPart(ctx context.Context, tenantID uint64, name, localPart string) (*entity.Sender, error)
	GetByID(ctx context.Context, tenantID, senderID uint64) (*entity.Sender, error)
}

type senderRepo struct {
	baseRepo BaseRepo
}

func NewSenderRepo(_ context.Context, baseRepo BaseRepo) SenderRepo {
	return &senderRepo{
		baseRepo: baseRepo,
	}
}

func (r *senderRepo) GetByID(ctx context.Context, tenantID, senderID uint64) (*entity.Sender, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "id",
			Value: senderID,
			Op:    OpEq,
		},
	})
}

func (r *senderRepo) GetManyByTenantID(ctx context.Context, tenantID uint64) ([]*entity.Sender, error) {
	return r.getMany(ctx, tenantID, nil)
}

func (r *senderRepo) GetByNameAndLocalPart(ctx context.Context, tenantID uint64, name, localPart string) (*entity.Sender, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "name",
			Value: name,
			Op:    OpEq,
		},
		{
			Field: "local_part",
			Value: localPart,
			Op:    OpEq,
		},
	})
}

func (r *senderRepo) get(ctx context.Context, tenantID uint64, conditions []*Condition) (*entity.Sender, error) {
	sender := new(Sender)

	if err := r.baseRepo.Get(ctx, sender, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), conditions...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSenderNotFound
		}
		return nil, err
	}

	return ToSender(sender), nil
}

func (r *senderRepo) getMany(ctx context.Context, tenantID uint64, conditions []*Condition) ([]*entity.Sender, error) {
	res, _, err := r.baseRepo.GetMany(ctx, new(Sender), &Filter{
		Conditions: append(r.getBaseConditions(tenantID), conditions...),
	})
	if err != nil {
		return nil, err
	}

	senders := make([]*entity.Sender, len(res))
	for i, m := range res {
		senders[i] = ToSender(m.(*Sender))
	}

	return senders, nil
}

func (r *senderRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field: "tenant_id",
			Value: tenantID,
			Op:    OpEq,
		},
	}
}

func (r *senderRepo) Create(ctx context.Context, sender *entity.Sender) (uint64, error) {
	senderModel := ToSenderModel(sender)

	if err := r.baseRepo.Create(ctx, senderModel); err != nil {
		return 0, err
	}

	return senderModel.GetID(), nil
}

func ToSender(sender *Sender) *entity.Sender {
	return &entity.Sender{
		ID:         sender.ID,
		TenantID:   sender.TenantID,
		Name:       sender.Name,
		LocalPart:  sender.LocalPart,
		CreateTime: sender.CreateTime,
		UpdateTime: sender.UpdateTime,
	}
}

func ToSenderModel(sender *entity.Sender) *Sender {
	return &Sender{
		ID:         sender.ID,
		TenantID:   sender.TenantID,
		Name:       sender.Name,
		LocalPart:  sender.LocalPart,
		CreateTime: sender.CreateTime,
		UpdateTime: sender.UpdateTime,
	}
}
