package handler

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"cdp/pkg/validator"
	"cdp/repo"
	"context"
	"github.com/rs/zerolog/log"
	"time"
)

type EmailHandler interface {
	CreateEmail(ctx context.Context, req *CreateEmailRequest, res *CreateEmailResponse) error
	GetEmail(ctx context.Context, req *GetEmailRequest, res *GetEmailResponse) error
	GetEmails(ctx context.Context, req *GetEmailsRequest, res *GetEmailsResponse) error
}

type emailHandler struct {
	emailRepo repo.EmailRepo
}

func NewEmailHandler(emailRepo repo.EmailRepo) EmailHandler {
	return &emailHandler{
		emailRepo: emailRepo,
	}
}

type CreateEmailRequest struct {
	Name      *string `json:"name,omitempty"`
	EmailDesc *string `json:"email_desc,omitempty"`
	Json      *string `json:"json,omitempty"`
	Html      *string `json:"html,omitempty"`
}

func (req *CreateEmailRequest) ToEmail() *entity.Email {
	now := time.Now()
	return &entity.Email{
		Name:       req.Name,
		EmailDesc:  req.EmailDesc,
		Json:       req.Json,
		Html:       req.Html,
		Status:     entity.EmailStatusNormal,
		CreateTime: goutil.Uint64(uint64(now.Unix())),
		UpdateTime: goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateEmailResponse struct {
	Email *entity.Email `json:"email,omitempty"`
}

var CreateEmailValidator = validator.MustForm(map[string]validator.Validator{
	"name":       ResourceNameValidator(false),
	"email_desc": ResourceDescValidator(false),
	"json":       &validator.String{},
	"html": &validator.String{
		Validators: []validator.StringFunc{goutil.IsBase64EncodedHTML},
	},
})

func (h *emailHandler) CreateEmail(ctx context.Context, req *CreateEmailRequest, res *CreateEmailResponse) error {
	if err := CreateEmailValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	email := req.ToEmail()
	id, err := h.emailRepo.Create(ctx, email)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create email failed: %v", err)
		return err
	}

	email.ID = goutil.Uint64(id)
	res.Email = email

	return nil
}

type GetEmailRequest struct {
	EmailID *uint64 `json:"email_id,omitempty"`
}

type GetEmailResponse struct {
	Email *entity.Email `json:"email,omitempty"`
}

var GetEmailValidator = validator.MustForm(map[string]validator.Validator{
	"email_id": &validator.UInt64{},
})

func (h *emailHandler) GetEmail(ctx context.Context, req *GetEmailRequest, res *GetEmailResponse) error {
	if err := GetEmailValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	email, err := h.emailRepo.Get(ctx, &repo.EmailFilter{
		ID: req.EmailID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get email err: %v", err)
		return err
	}

	res.Email = email

	return nil
}

type GetEmailsRequest struct {
	Name       *string            `json:"name,omitempty"`
	EmailDesc  *string            `json:"email_desc,omitempty"`
	Pagination *entity.Pagination `json:"pagination,omitempty"`
}

type GetEmailsResponse struct {
	Emails     []*entity.Email    `json:"emails"`
	Pagination *entity.Pagination `json:"pagination,omitempty"`
}

var GetEmailsValidator = validator.MustForm(map[string]validator.Validator{
	"name":       ResourceNameValidator(true),
	"email_desc": ResourceDescValidator(true),
	"pagination": PaginationValidator(),
})

func (h *emailHandler) GetEmails(ctx context.Context, req *GetEmailsRequest, res *GetEmailsResponse) error {
	if err := GetEmailsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.Pagination == nil {
		req.Pagination = new(entity.Pagination)
	}

	emails, pagination, err := h.emailRepo.GetMany(ctx, &repo.EmailFilter{
		Name:      req.Name,
		EmailDesc: req.EmailDesc,
		Pagination: &repo.Pagination{
			Page:  req.Pagination.Page,
			Limit: req.Pagination.Limit,
		},
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get emails failed: %v", err)
		return err
	}

	res.Emails = emails
	res.Pagination = pagination

	return nil
}
