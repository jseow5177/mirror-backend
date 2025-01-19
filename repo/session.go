package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"context"
	"errors"
	"gorm.io/gorm"
	"time"
)

var (
	ErrSessionNotFound = errutil.NotFoundError(errors.New("session not found"))
)

type Session struct {
	ID         *uint64
	UserID     *uint64
	TokenHash  *string
	ExpireTime *uint64
	CreateTime *uint64
}

func (m *Session) TableName() string {
	return "session_tab"
}

func (m *Session) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

type SessionRepo interface {
	Create(ctx context.Context, session *entity.Session) (uint64, error)
	GetByUserID(ctx context.Context, userID uint64) (*entity.Session, error)
	GetByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error)
}

type sessionRepo struct {
	baseRepo BaseRepo
}

func NewSessionRepo(_ context.Context, baseRepo BaseRepo) SessionRepo {
	return &sessionRepo{baseRepo: baseRepo}
}

func (r *sessionRepo) GetByUserID(ctx context.Context, userID uint64) (*entity.Session, error) {
	return r.get(ctx, []*Condition{
		{
			Field: "user_id",
			Value: userID,
			Op:    OpEq,
		},
	}, true)
}

func (r *sessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error) {
	return r.get(ctx, []*Condition{
		{
			Field: "token_hash",
			Value: tokenHash,
			Op:    OpEq,
		},
	}, true)
}

func (r *sessionRepo) get(ctx context.Context, conditions []*Condition, filterExpire bool) (*entity.Session, error) {
	session := new(Session)

	if err := r.baseRepo.Get(ctx, session, &Filter{
		Conditions: r.maybeAddExpireFilter(conditions, filterExpire),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, err
	}

	return ToSession(session), nil
}

func (r *sessionRepo) maybeAddExpireFilter(conditions []*Condition, filterExpire bool) []*Condition {
	if filterExpire {
		return append(conditions, &Condition{
			Field: "expire_time",
			Value: time.Now().Unix(),
			Op:    OpLte,
		})
	}
	return conditions
}

func (r *sessionRepo) Create(ctx context.Context, session *entity.Session) (uint64, error) {
	sessionModel := ToSessionModel(session)

	if err := r.baseRepo.Create(ctx, sessionModel); err != nil {
		return 0, err
	}

	return sessionModel.GetID(), nil
}

func ToSession(session *Session) *entity.Session {
	return &entity.Session{
		ID:         session.ID,
		UserID:     session.UserID,
		Token:      nil,
		TokenHash:  session.TokenHash,
		ExpireTime: session.ExpireTime,
		CreateTime: session.CreateTime,
	}
}

func ToSessionModel(session *entity.Session) *Session {
	return &Session{
		ID:         session.ID,
		UserID:     session.UserID,
		TokenHash:  session.TokenHash,
		ExpireTime: session.ExpireTime,
		CreateTime: session.CreateTime,
	}
}
