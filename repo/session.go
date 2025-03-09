package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
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
	DeleteByUserID(ctx context.Context, userID uint64) error
}

type sessionRepo struct {
	cacheKeyPrefix string
	baseRepo       BaseRepo
	baseCache      BaseCache
}

func NewSessionRepo(ctx context.Context, baseRepo BaseRepo) (SessionRepo, error) {
	r := &sessionRepo{
		cacheKeyPrefix: "session",
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
					log.Ctx(ctx).Error().Msgf("failed to refresh session cache, err: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return r, nil
}

func (r *sessionRepo) refreshCache(ctx context.Context) error {
	allSessions, err := r.getMany(ctx, nil, true)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msgf("failed to refresh session cache, err: %v", err)
		return err
	}

	r.baseCache.Flush(ctx)

	log.Ctx(ctx).Info().Msgf("refreshing session cache, %d sessions found", len(allSessions))
	for _, session := range allSessions {
		r.setCache(ctx, session)
	}

	return nil
}

func (r *sessionRepo) setCache(ctx context.Context, session *entity.Session) {
	r.baseCache.Set(ctx, r.cacheKeyPrefix, 0, session.GetTokenHash(), session)
	r.baseCache.Set(ctx, r.cacheKeyPrefix, 0, session.GetUserID(), session)
}

func (r *sessionRepo) getFromCache(ctx context.Context, uniqKey interface{}) *entity.Session {
	if v, ok := r.baseCache.Get(ctx, r.cacheKeyPrefix, 0, uniqKey); ok {
		return v.(*entity.Session)
	}
	return nil
}

func (r *sessionRepo) deleteFromCache(ctx context.Context, uniqKey interface{}) {
	r.baseCache.Del(ctx, r.cacheKeyPrefix, 0, uniqKey)
}

func (r *sessionRepo) GetByUserID(ctx context.Context, userID uint64) (*entity.Session, error) {
	session := r.getFromCache(ctx, userID)
	if session != nil {
		return session, nil
	}

	return r.get(ctx, []*Condition{
		{
			Field: "user_id",
			Value: userID,
			Op:    OpEq,
		},
	}, true)
}

func (r *sessionRepo) GetByTokenHash(ctx context.Context, tokenHash string) (*entity.Session, error) {
	session := r.getFromCache(ctx, tokenHash)
	if session != nil {
		return session, nil
	}

	return r.get(ctx, []*Condition{
		{
			Field: "token_hash",
			Value: tokenHash,
			Op:    OpEq,
		},
	}, true)
}

func (r *sessionRepo) DeleteByUserID(ctx context.Context, userID uint64) error {
	if err := r.baseRepo.Delete(ctx, new(Session), &Filter{
		Conditions: []*Condition{
			{
				Field: "user_id",
				Value: userID,
				Op:    OpEq,
			},
		},
	}); err != nil {
		return err
	}

	session := r.getFromCache(ctx, userID)
	if session != nil {
		r.deleteFromCache(ctx, userID)
		r.deleteFromCache(ctx, session.GetTokenHash())
	}

	return nil
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

func (r *sessionRepo) getMany(ctx context.Context, conditions []*Condition, filterExpire bool) ([]*entity.Session, error) {
	res, _, err := r.baseRepo.GetMany(ctx, new(Session), &Filter{
		Conditions: r.maybeAddExpireFilter(conditions, filterExpire),
	})
	if err != nil {
		return nil, err
	}

	sessions := make([]*entity.Session, len(res))
	for i, m := range res {
		sessions[i] = ToSession(m.(*Session))
	}

	return sessions, nil
}

func (r *sessionRepo) maybeAddExpireFilter(conditions []*Condition, filterExpire bool) []*Condition {
	if filterExpire {
		return append(conditions, &Condition{
			Field: "expire_time",
			Value: time.Now().Add(config.ThreeMonths).Unix(),
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
