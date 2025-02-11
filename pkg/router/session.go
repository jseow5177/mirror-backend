package router

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/httputil"
	"cdp/repo"
	"context"
	"errors"
	"github.com/patrickmn/go-cache"
	"github.com/rs/zerolog/log"
	"math/rand"
	"net/http"
	"time"
)

type userTenantCache struct {
	user   *entity.User
	tenant *entity.Tenant
}

var c *cache.Cache

func init() {
	c = cache.New(30*time.Minute, 15*time.Minute)
}

type ContextInfo interface {
	SetUser(user *entity.User)
	SetTenant(tenant *entity.Tenant)
}

type contextKey string

const (
	userKey   contextKey = "user"
	tenantKey contextKey = "tenant"
)

type sessionMiddleware struct {
	userRepo    repo.UserRepo
	tenantRepo  repo.TenantRepo
	sessionRepo repo.SessionRepo
}

func NewSessionMiddleware(userRepo repo.UserRepo, tenantRepo repo.TenantRepo, sessionRepo repo.SessionRepo) Middleware {
	return &sessionMiddleware{
		userRepo:    userRepo,
		tenantRepo:  tenantRepo,
		sessionRepo: sessionRepo,
	}
}

func (m *sessionMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var token string
		for _, cookie := range r.Cookies() {
			if cookie.Name == "session" {
				token = cookie.Value
				break
			}
		}
		if token == "" {
			log.Ctx(ctx).Error().Msg("token is empty")
			m.returnErr(w)
			return
		}

		var utc *userTenantCache
		if c != nil {
			v, ok := c.Get(token)
			if ok {
				utc = v.(*userTenantCache)
			}
		}

		if utc == nil {
			decodedToken, err := goutil.Base64Decode(token)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("decode token error, err: %v", err)
				m.returnErr(w)
				return
			}

			session, err := m.sessionRepo.GetByTokenHash(ctx, goutil.Sha256(decodedToken))
			if err != nil {
				log.Ctx(ctx).Error().Msgf("get session error, err: %v", err)
				m.returnErr(w)
				return
			}

			user, err := m.userRepo.GetByID(ctx, session.GetUserID())
			if err != nil {
				log.Ctx(ctx).Error().Msgf("get user error, err: %v, userID: %v", err, session.GetUserID())
				m.returnErr(w)
				return
			}

			tenant, err := m.tenantRepo.GetByID(ctx, user.GetTenantID())
			if err != nil {
				log.Ctx(ctx).Error().Msgf("get tenant error, err: %v, tenantID: %v", err, user.GetTenantID())
				m.returnErr(w)
				return
			}

			utc = &userTenantCache{
				user:   user,
				tenant: tenant,
			}

			if r.URL.String() == getAppPath(config.PathLogOut) {
				c.Delete(token)
			} else {
				exp := time.Duration(rand.Intn(10))*time.Minute + 30*time.Minute
				c.Set(token, utc, exp)
			}
		}

		ctx = context.WithValue(ctx, userKey, utc.user)
		ctx = context.WithValue(ctx, tenantKey, utc.tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *sessionMiddleware) returnErr(w http.ResponseWriter) {
	// abstract all errors as invalid session
	httputil.ReturnServerResponse(w, nil, errutil.UnauthorizedError(errors.New("invalid session")))
}

func GetUserFromContext(ctx context.Context) (*entity.User, bool) {
	val := ctx.Value(userKey)
	if user, ok := val.(*entity.User); ok {
		return user, true
	}
	return nil, false
}

func GetTenantFromContext(ctx context.Context) (*entity.Tenant, bool) {
	val := ctx.Value(tenantKey)
	if tenant, ok := val.(*entity.Tenant); ok {
		return tenant, true
	}
	return nil, false
}
