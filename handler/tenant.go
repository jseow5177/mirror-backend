package handler

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"time"
)

type TenantHandler interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error
	GetTenant(ctx context.Context, req *GetTenantRequest, res *GetTenantResponse) error
}

type tenantHandler struct {
	cfg         *config.Config
	txService   repo.TxService
	tenantRepo  repo.TenantRepo
	fileRepo    repo.FileRepo
	queryRepo   repo.QueryRepo
	userHandler UserHandler
}

func NewTenantHandler(
	cfg *config.Config,
	txService repo.TxService,
	tenantRepo repo.TenantRepo,
	fileRepo repo.FileRepo,
	queryRepo repo.QueryRepo,
	userHandler UserHandler,
) TenantHandler {
	return &tenantHandler{
		cfg:         cfg,
		txService:   txService,
		tenantRepo:  tenantRepo,
		fileRepo:    fileRepo,
		queryRepo:   queryRepo,
		userHandler: userHandler,
	}
}

type CreateTenantRequest struct {
	Name  *string              `json:"name,omitempty"`
	Users []*CreateUserRequest `json:"users,omitempty"`
}

func (r *CreateTenantRequest) GetName() string {
	if r != nil && r.Name != nil {
		return *r.Name
	}
	return ""
}

func (r *CreateTenantRequest) ToTenant(folderID string) *entity.Tenant {
	now := uint64(time.Now().Unix())

	return &entity.Tenant{
		Name:   r.Name,
		Status: entity.TenantStatusNormal,
		ExtInfo: &entity.TenantExtInfo{
			FolderID: folderID,
		},
		CreateTime: goutil.Uint64(now),
		UpdateTime: goutil.Uint64(now),
	}
}

type CreateTenantResponse struct {
	Tenant *entity.Tenant `json:"tenant,omitempty"`
	Users  []*entity.User `json:"users,omitempty"`
}

var CreateTenantValidator = validator.MustForm(map[string]validator.Validator{
	"name": ResourceNameValidator(false),
})

func (h *tenantHandler) CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error {
	if err := CreateTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	_, err := h.tenantRepo.GetByName(ctx, req.GetName())
	if err == nil {
		return errutil.ConflictError(errors.New("tenant already exists"))
	}

	if !errors.Is(err, repo.ErrTenantNotFound) {
		log.Ctx(ctx).Error().Msgf("get tenant failed: %v", err)
		return err
	}

	if err := h.txService.RunTx(ctx, func(ctx context.Context) error {
		// create file store
		folderID, err := h.fileRepo.CreateFolder(ctx, req.GetName())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create tenant folder failed: %v, tenant: %v", err, req.GetName())
			return err
		}

		tenant := req.ToTenant(folderID)

		// create tenant
		id, err := h.tenantRepo.Create(ctx, tenant)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create tenant failed: %v", err)
			return err
		}

		tenant.ID = goutil.Uint64(id)

		// create query store
		if err := h.queryRepo.CreateStore(ctx, tenant.GetName()); err != nil {
			log.Ctx(ctx).Error().Msgf("create query store failed: %v", err)
			return err
		}

		// create users
		var (
			contextInfo = ContextInfo{
				Tenant: tenant,
			}
			createUsersReq = &CreateUsersRequest{
				ContextInfo: contextInfo,
				Users:       req.Users,
			}
			createUsersRes = new(CreateUsersResponse)
		)
		if err := h.userHandler.CreateUsers(ctx, createUsersReq, createUsersRes); err != nil {
			log.Ctx(ctx).Error().Msgf("create users failed: %v", err)
			return err
		}

		tenant.ID = goutil.Uint64(id)
		res.Tenant = tenant

		return nil
	}); err != nil {
		return err
	}

	return nil
}

type GetTenantRequest struct {
	ContextInfo
}

type GetTenantResponse struct {
	Tenant *entity.Tenant `json:"tenant,omitempty"`
}

var GetTenantValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, true),
})

func (h *tenantHandler) GetTenant(_ context.Context, req *GetTenantRequest, res *GetTenantResponse) error {
	if err := GetTenantValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	res.Tenant = req.Tenant

	return nil
}
