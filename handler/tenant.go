package handler

import (
	"cdp/config"
	"cdp/dep"
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"errors"
	"github.com/rs/zerolog/log"
	"regexp"
	"time"
)

type TenantHandler interface {
	CreateTenant(ctx context.Context, req *CreateTenantRequest, res *CreateTenantResponse) error
	GetTenant(ctx context.Context, req *GetTenantRequest, res *GetTenantResponse) error
	CreateDomain(ctx context.Context, req *CreateDomainRequest, res *CreateDomainResponse) error
	UpdateDnsRecords(ctx context.Context, req *UpdateDnsRecordsRequest, res *UpdateDnsRecordsResponse) error
	CreateSender(ctx context.Context, req *CreateSenderRequest, res *CreateSenderResponse) error
	GetSenders(ctx context.Context, req *GetSendersRequest, res *GetSendersResponse) error
}

type tenantHandler struct {
	cfg          *config.Config
	txService    repo.TxService
	tenantRepo   repo.TenantRepo
	fileRepo     repo.FileRepo
	queryRepo    repo.QueryRepo
	roleRepo     repo.RoleRepo
	userRoleRepo repo.UserRoleRepo
	userHandler  UserHandler
	emailService dep.EmailService
	senderRepo   repo.SenderRepo
}

func NewTenantHandler(
	cfg *config.Config,
	txService repo.TxService,
	tenantRepo repo.TenantRepo,
	fileRepo repo.FileRepo,
	queryRepo repo.QueryRepo,
	roleRepo repo.RoleRepo,
	userRoleRepo repo.UserRoleRepo,
	userHandler UserHandler,
	emailService dep.EmailService,
	senderRepo repo.SenderRepo,
) TenantHandler {
	return &tenantHandler{
		cfg:          cfg,
		txService:    txService,
		tenantRepo:   tenantRepo,
		fileRepo:     fileRepo,
		queryRepo:    queryRepo,
		roleRepo:     roleRepo,
		userRoleRepo: userRoleRepo,
		userHandler:  userHandler,
		emailService: emailService,
		senderRepo:   senderRepo,
	}
}

type CreateTenantRequest struct {
	Name   *string              `json:"name,omitempty"`
	Users  []*CreateUserRequest `json:"users,omitempty"`
	Domain *string              `json:"domain,omitempty"`
}

func (r *CreateTenantRequest) GetName() string {
	if r != nil && r.Name != nil {
		return *r.Name
	}
	return ""
}

func (r *CreateTenantRequest) GetDomain() string {
	if r != nil && r.Domain != nil {
		return *r.Domain
	}
	return ""
}

func (r *CreateTenantRequest) ToTenant(folderID string, dnsRecords map[string]map[string]interface{}) *entity.Tenant {
	now := uint64(time.Now().Unix())

	return &entity.Tenant{
		Name:   r.Name,
		Status: entity.TenantStatusNormal,
		ExtInfo: &entity.TenantExtInfo{
			FolderID:      folderID,
			Domain:        r.GetDomain(),
			DnsRecords:    dnsRecords,
			IsDomainValid: goutil.Bool(false),
		},
		CreateTime: goutil.Uint64(now),
		UpdateTime: goutil.Uint64(now),
	}
}

type CreateTenantResponse struct {
	Tenant     *entity.Tenant                    `json:"tenant,omitempty"`
	Users      []*entity.User                    `json:"users,omitempty"`
	DNSRecords map[string]map[string]interface{} `json:"dns_records,omitempty"`
}

var CreateTenantValidator = validator.MustForm(map[string]validator.Validator{
	"name":   ResourceNameValidator(false),
	"domain": DomainValidator(true),
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
		// create domain
		var dnsRecords map[string]map[string]interface{}
		if req.GetDomain() != "" {
			var err error
			dnsRecords, err = h.emailService.CreateDomain(ctx, req.GetDomain())
			if err != nil {
				log.Ctx(ctx).Error().Msgf("create domain failed: %v", err)
				return err
			}
		}

		// create file store
		folderID, err := h.fileRepo.CreateFolder(ctx, req.GetName())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create tenant folder failed: %v, tenant: %v", err, req.GetName())
			return err
		}

		tenant := req.ToTenant(folderID, dnsRecords)

		// create tenant
		tenantID, err := h.tenantRepo.Create(ctx, tenant)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create tenant failed: %v", err)
			return err
		}

		tenant.ID = goutil.Uint64(tenantID)

		// create query store
		if err := h.queryRepo.CreateStore(ctx, tenant.GetName()); err != nil {
			log.Ctx(ctx).Error().Msgf("create query store failed: %v", err)
			return err
		}

		// create roles
		var defaultRoles = []*entity.Role{
			{
				ID:       nil,
				Name:     goutil.String("Admin"),
				RoleDesc: goutil.String("Admin role"),
				Status:   entity.RoleStatusNormal,
				Actions: []entity.ActionCode{
					entity.ActionEditRole,
					entity.ActionEditUser,
				},
				TenantID:   tenant.ID,
				CreateTime: tenant.CreateTime,
				UpdateTime: tenant.UpdateTime,
			},
			{
				ID:         nil,
				Name:       goutil.String("Member"),
				RoleDesc:   goutil.String("Member role"),
				Status:     entity.RoleStatusNormal,
				Actions:    []entity.ActionCode{},
				TenantID:   tenant.ID,
				CreateTime: tenant.CreateTime,
				UpdateTime: tenant.UpdateTime,
			},
		}

		roleIDs, err := h.roleRepo.CreateMany(ctx, defaultRoles)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("create roles failed: %v", err)
			return err
		}

		// set role IDs
		for _, user := range req.Users {
			user.RoleID = goutil.Uint64(roleIDs[0]) // set to Admin
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

		tenant.ID = goutil.Uint64(tenantID)
		res.Tenant = tenant
		res.Users = createUsersRes.Users

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

type CreateDomainRequest struct {
	ContextInfo

	Domain *string `json:"domain,omitempty"`
}

func (req *CreateDomainRequest) GetDomain() string {
	if req != nil && req.Domain != nil {
		return *req.Domain
	}
	return ""
}

type CreateDomainResponse struct {
	DNSRecords map[string]map[string]interface{} `json:"dns_records,omitempty"`
}

var CreateDomainValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
	"domain":      DomainValidator(false),
})

func (h *tenantHandler) CreateDomain(ctx context.Context, req *CreateDomainRequest, res *CreateDomainResponse) error {
	if err := CreateDomainValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	dnsRecords, err := h.emailService.CreateDomain(ctx, req.GetDomain())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create domain failed: %v", err)
		return err
	}

	req.Tenant.Update(&entity.Tenant{
		ExtInfo: &entity.TenantExtInfo{
			Domain:     req.GetDomain(),
			DnsRecords: dnsRecords,
		},
	})

	if err := h.tenantRepo.Update(ctx, req.Tenant); err != nil {
		log.Ctx(ctx).Error().Msgf("update tenant failed: %v", err)
		return err
	}

	res.DNSRecords = dnsRecords

	return nil
}

type CreateSenderRequest struct {
	ContextInfo

	Name      *string `json:"name,omitempty"`
	LocalPart *string `json:"local_part,omitempty"`
}

func (r *CreateSenderRequest) ToSender() *entity.Sender {
	now := uint64(time.Now().Unix())
	return &entity.Sender{
		TenantID:   r.Tenant.ID,
		Name:       r.Name,
		LocalPart:  r.LocalPart,
		CreateTime: goutil.Uint64(now),
		UpdateTime: goutil.Uint64(now),
	}
}

func (r *CreateSenderRequest) GetName() string {
	if r != nil && r.Name != nil {
		return *r.Name
	}
	return ""
}

func (r *CreateSenderRequest) GetLocalPart() string {
	if r != nil && r.LocalPart != nil {
		return *r.LocalPart
	}
	return ""
}

type CreateSenderResponse struct {
	Sender *entity.Sender `json:"sender,omitempty"`
}

var CreateSenderValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
	"name": &validator.String{
		MaxLen: 64,
	},
	"local_part": &validator.String{
		MaxLen: 64,
		Validators: []validator.StringFunc{
			func(s string) error {
				re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+$`)
				if !re.MatchString(s) {
					return errors.New("invalid email local part")
				}
				return nil
			},
		},
	},
})

func (h *tenantHandler) CreateSender(ctx context.Context, req *CreateSenderRequest, res *CreateSenderResponse) error {
	if err := CreateSenderValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.Tenant.ExtInfo.GetDomain() == "" || !req.Tenant.ExtInfo.GetIsDomainValid() {
		return errutil.ValidationError(errors.New("domain is not verified"))
	}

	_, err := h.senderRepo.GetByNameAndLocalPart(ctx, req.GetTenantID(), req.GetName(), req.GetLocalPart())
	if err == nil {
		return errutil.ConflictError(errors.New("sender already exists"))
	}

	if !errors.Is(err, repo.ErrSenderNotFound) {
		log.Ctx(ctx).Error().Msgf("get sender failed: %v", err)
		return err
	}

	var (
		sender = req.ToSender()
		email  = sender.GetEmail(req.Tenant)
	)
	sender.Email = goutil.String(email)

	if err := h.emailService.CreateSender(ctx, req.GetName(), email); err != nil {
		log.Ctx(ctx).Error().Msgf("create sender in email service failed: %v", err)
		return err
	}

	id, err := h.senderRepo.Create(ctx, sender)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create sender in db failed: %v", err)
		return err
	}

	sender.ID = goutil.Uint64(id)
	res.Sender = sender

	return nil
}

type GetSendersRequest struct {
	ContextInfo
}

type GetSendersResponse struct {
	Senders []*entity.Sender `json:"senders"`
}

var GetSendersValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
})

func (h *tenantHandler) GetSenders(ctx context.Context, req *GetSendersRequest, res *GetSendersResponse) error {
	if err := GetSendersValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	senders, err := h.senderRepo.GetManyByTenantID(ctx, req.GetTenantID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get senders failed: %v", err)
		return err
	}

	for _, sender := range senders {
		sender.Email = goutil.String(sender.GetEmail(req.Tenant))
	}

	res.Senders = senders

	return nil
}

type UpdateDnsRecordsRequest struct {
	ContextInfo
}

type UpdateDnsRecordsResponse struct {
	IsVerified *bool                             `json:"is_verified,omitempty"`
	DnsRecords map[string]map[string]interface{} `json:"dns_records,omitempty"`
}

var UpdateDnsRecordsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(false, false),
})

func (h *tenantHandler) UpdateDnsRecords(ctx context.Context, req *UpdateDnsRecordsRequest, res *UpdateDnsRecordsResponse) error {
	if err := UpdateDnsRecordsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	domain := req.Tenant.ExtInfo.GetDomain()
	if domain == "" {
		return errutil.ValidationError(errors.New("domain is empty"))
	}

	dnsRecords, err := h.emailService.GetDomainConfig(ctx, domain)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get domain config failed: %v", err)
		return err
	}

	// no need to return error
	isVerified, err := h.emailService.AuthenticateDomain(ctx, domain)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("authenticate domain failed: %v", err)
	}

	req.Tenant.Update(&entity.Tenant{
		ExtInfo: &entity.TenantExtInfo{
			IsDomainValid: goutil.Bool(isVerified),
			DnsRecords:    dnsRecords,
		},
	})

	if err := h.tenantRepo.Update(ctx, req.Tenant); err != nil {
		log.Ctx(ctx).Error().Msgf("update tenant failed: %v", err)
		return err
	}

	res.IsVerified = goutil.Bool(isVerified)
	res.DnsRecords = dnsRecords

	return nil
}
