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
	ErrTokenNotFound = errutil.NotFoundError(errors.New("token not found"))
)

type Activation struct {
	ID         *uint64
	TokenHash  *string
	TargetID   *uint64
	TokenType  *uint32
	ExpireTime *uint64
	CreateTime *uint64
}

func (m *Activation) TableName() string {
	return "activation_tab"
}

func (m *Activation) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

func (m *Activation) GetTokenType() uint32 {
	if m != nil && m.TokenType != nil {
		return *m.TokenType
	}
	return 0
}

type ActivationRepo interface {
	Create(ctx context.Context, act *entity.Activation) (uint64, error)
	Update(ctx context.Context, act *entity.Activation) error
	GetByTokenHash(ctx context.Context, tokenHash string, tokenType entity.TokenType) (*entity.Activation, error)
}

type activationRepo struct {
	baseRepo BaseRepo
}

func NewActivationRepo(_ context.Context, baseRepo BaseRepo) ActivationRepo {
	return &activationRepo{baseRepo: baseRepo}
}

func (r *activationRepo) GetByTokenHash(ctx context.Context, tokenHash string, tokenType entity.TokenType) (*entity.Activation, error) {
	return r.get(ctx, []*Condition{
		{
			Field:         "token_hash",
			Value:         tokenHash,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
		{
			Field: "token_type",
			Value: tokenType,
			Op:    OpEq,
		},
	})
}

func (r *activationRepo) get(ctx context.Context, conditions []*Condition) (*entity.Activation, error) {
	act := new(Activation)

	if err := r.baseRepo.Get(ctx, act, &Filter{
		Conditions: r.baseRepo.BuildConditions(r.getBaseConditions(), conditions),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenNotFound
		}
		return nil, err
	}

	return ToActivation(act), nil
}

func (r *activationRepo) getBaseConditions() []*Condition {
	// TODO: Expiry logic
	return []*Condition{}
}

func (r *activationRepo) Create(ctx context.Context, act *entity.Activation) (uint64, error) {
	actModel := ToActivationModel(act)

	if err := r.baseRepo.Create(ctx, actModel); err != nil {
		return 0, err
	}

	return actModel.GetID(), nil
}

func (r *activationRepo) Update(ctx context.Context, act *entity.Activation) error {
	if err := r.baseRepo.Update(ctx, ToActivationModel(act)); err != nil {
		return err
	}

	return nil
}

func ToActivation(act *Activation) *entity.Activation {
	return &entity.Activation{
		ID:         act.ID,
		TokenHash:  act.TokenHash,
		TargetID:   act.TargetID,
		TokenType:  entity.TokenType(act.GetTokenType()),
		ExpireTime: act.ExpireTime,
		CreateTime: act.CreateTime,
	}
}

func ToActivationModel(act *entity.Activation) *Activation {
	return &Activation{
		ID:         act.ID,
		TokenHash:  act.TokenHash,
		TargetID:   act.TargetID,
		TokenType:  goutil.Uint32(uint32(act.GetTokenType())),
		CreateTime: act.CreateTime,
		ExpireTime: act.ExpireTime,
	}
}
