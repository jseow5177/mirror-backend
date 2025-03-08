package router

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/httputil"
	"cdp/repo"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"net/http"
)

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
	userRepo       repo.UserRepo
	tenantRepo     repo.TenantRepo
	sessionRepo    repo.SessionRepo
	roleRepo       repo.RoleRepo
	userRoleRepo   repo.UserRoleRepo
	allowedActions []entity.ActionCode
}

func NewSessionMiddleware(userRepo repo.UserRepo, tenantRepo repo.TenantRepo,
	sessionRepo repo.SessionRepo, roleRepo repo.RoleRepo, userRoleRepo repo.UserRoleRepo, allowedActions []entity.ActionCode) Middleware {
	return &sessionMiddleware{
		userRepo:       userRepo,
		tenantRepo:     tenantRepo,
		sessionRepo:    sessionRepo,
		roleRepo:       roleRepo,
		userRoleRepo:   userRoleRepo,
		allowedActions: allowedActions,
	}
}

func (m *sessionMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var (
			invalidSessionErr = errutil.UnauthorizedError(errors.New("invalid session"))
			forbiddenErr      = errutil.ForbiddenError(errors.New("unauthorized"))
		)

		var token string
		for _, cookie := range r.Cookies() {
			if cookie.Name == "session" {
				token = cookie.Value
				break
			}
		}
		if token == "" {
			log.Ctx(ctx).Error().Msg("token is empty")
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		decodedToken, err := goutil.Base64Decode(token)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("decode token error, err: %v", err)
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		session, err := m.sessionRepo.GetByTokenHash(ctx, goutil.Sha256(decodedToken))
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get session error, err: %v", err)
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		user, err := m.userRepo.GetByID(ctx, session.GetUserID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get user error, err: %v, userID: %v", err, session.GetUserID())
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		tenant, err := m.tenantRepo.GetByID(ctx, user.GetTenantID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get tenant error, err: %v, tenantID: %v", err, user.GetTenantID())
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		userRoleRepo, err := m.userRoleRepo.GetByUserID(ctx, user.GetTenantID(), user.GetID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get user role error, err: %v, userID: %v", err, user.GetID())
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		role, err := m.roleRepo.GetByID(ctx, user.GetTenantID(), userRoleRepo.GetRoleID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get user role error, err: %v, userID: %v", err, user.GetID())
			httputil.ReturnServerResponse(w, nil, invalidSessionErr)
			return
		}

		allowed := len(m.allowedActions) == 0
		if !allowed {
			actionSet := make(map[entity.ActionCode]struct{})
			for _, action := range m.allowedActions {
				actionSet[action] = struct{}{}
			}

			for _, actionCode := range role.Actions {
				if _, exists := actionSet[actionCode]; exists {
					allowed = true
					break
				}
			}
		}

		if !allowed {
			httputil.ReturnServerResponse(w, nil, forbiddenErr)
			return
		}

		user.Role = role

		ctx = context.WithValue(ctx, userKey, user)
		ctx = context.WithValue(ctx, tenantKey, tenant)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
