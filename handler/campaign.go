package handler

import (
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

type CampaignHandler interface {
	CreateCampaign(ctx context.Context, req *CreateCampaignRequest, res *CreateCampaignResponse) error
	OnEmailOpen(ctx context.Context, req *OnEmailOpenRequest, res *OnEmailOpenResponse) error
	OnEmailButtonClick(ctx context.Context, req *OnEmailButtonClickRequest, res *OnEmailButtonClickResponse) error
}

type campaignHandler struct {
	campaignRepo   repo.CampaignRepo
	emailRepo      repo.EmailRepo
	segmentHandler SegmentHandler
}

func NewCampaignHandler(campaignRepo repo.CampaignRepo, emailRepo repo.EmailRepo, segmentHandler SegmentHandler) CampaignHandler {
	return &campaignHandler{
		campaignRepo,
		emailRepo,
		segmentHandler,
	}
}

type CampaignEmail struct {
	EmailID *uint64 `json:"email_id,omitempty"`
	Subject *string `json:"subject,omitempty"`
	Ratio   *uint64 `json:"ratio,omitempty"`
}

func (e *CampaignEmail) GetRatio() uint64 {
	if e != nil && e.Ratio != nil {
		return *e.Ratio
	}
	return 0
}

type CreateCampaignRequest struct {
	Name         *string          `json:"name,omitempty"`
	CampaignDesc *string          `json:"campaign_desc,omitempty"`
	SegmentID    *uint64          `json:"segment_id,omitempty"`
	Emails       []*CampaignEmail `json:"emails,omitempty"`
	Schedule     *uint64          `json:"schedule,omitempty"`
}

func (r *CreateCampaignRequest) GetSchedule() uint64 {
	if r != nil && r.Schedule != nil {
		return *r.Schedule
	}
	return 0
}

type CreateCampaignResponse struct {
	Campaign *entity.Campaign `json:"campaign"`
}

var CreateCampaignValidator = validator.MustForm(map[string]validator.Validator{
	"name":          ResourceNameValidator(false),
	"campaign_desc": ResourceDescValidator(false),
	"segment_id":    &validator.UInt64{},
	"emails": &validator.Slice{
		MinLen: 1,
		MaxLen: 2,
		Validator: validator.MustForm(map[string]validator.Validator{
			"email_id": &validator.UInt64{},
			"subject": &validator.String{
				MinLen: 1,
				MaxLen: 100,
			},
			"ratio": &validator.UInt64{
				Optional: true,
			},
		}),
	},
	"schedule": &validator.UInt64{
		Optional: true,
	},
})

func (h *campaignHandler) CreateCampaign(ctx context.Context, req *CreateCampaignRequest, res *CreateCampaignResponse) error {
	if err := CreateCampaignValidator.Validate(req); err != nil {
		return err
	}

	// validate ratio
	var ratio uint64
	for _, email := range req.Emails {
		ratio += email.GetRatio()
	}
	if ratio != 100 {
		return errutil.ValidationError(errors.New("ratios must add up to 100"))
	}

	htmls := make([]string, 0)
	campaignEmails := make([]*entity.CampaignEmail, 0, len(req.Emails))
	for _, email := range req.Emails {
		e, err := h.emailRepo.Get(ctx, &repo.EmailFilter{
			ID: email.EmailID,
		})
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get email err: %v", err)
			return err
		}
		campaignEmails = append(campaignEmails, &entity.CampaignEmail{
			EmailID:     email.EmailID,
			Subject:     email.Subject,
			Html:        goutil.String(""),
			Ratio:       email.Ratio,
			OpenCount:   goutil.Uint64(uint64(0)),
			ClickCounts: make(map[string]uint64),
		})

		htmls = append(htmls, e.GetHtml())
	}

	// create campaign
	now := time.Now()
	campaign := &entity.Campaign{
		Name:           req.Name,
		CampaignDesc:   req.CampaignDesc,
		SegmentID:      req.SegmentID,
		SegmentSize:    goutil.Uint64(0),
		Progress:       goutil.Uint64(0),
		CampaignEmails: campaignEmails,
		Status:         entity.CampaignStatusPending,
		Schedule:       goutil.Uint64(req.GetSchedule()),
		CreateTime:     goutil.Uint64(uint64(now.Unix())),
		UpdateTime:     goutil.Uint64(uint64(now.Unix())),
	}

	_, err := h.campaignRepo.Create(ctx, campaign)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign failed: %v", err)
		return err
	}

	res.Campaign = campaign

	return nil
}

type OnEmailOpenRequest struct {
	CampaignEmailID *uint64 `schema:"campaign_email_id,omitempty"`
}

type OnEmailOpenResponse struct{}

var OnEmailOpenValidator = validator.MustForm(map[string]validator.Validator{
	"campaign_email_id": &validator.UInt64{},
})

func (h *campaignHandler) OnEmailOpen(ctx context.Context, req *OnEmailOpenRequest, res *OnEmailOpenResponse) error {
	if err := OnEmailOpenValidator.Validate(req); err != nil {
		return err
	}

	campaignEmail, err := h.campaignRepo.GetCampaignEmail(ctx, &repo.CampaignEmailFilter{
		ID: req.CampaignEmailID,
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaign email err: %v", err)
		return err
	}

	f := &repo.CampaignEmailFilter{
		ID: req.CampaignEmailID,
	}

	if err := h.campaignRepo.UpdateCampaignEmail(ctx, f, &entity.CampaignEmail{
		OpenCount: goutil.Uint64(campaignEmail.GetOpenCount() + 1),
	}); err != nil {
		log.Ctx(ctx).Error().Msgf("update campaign email err: %v, campaign_email_id: %v", err, req.CampaignEmailID)
		return err
	}

	return nil
}

type OnEmailButtonClickRequest struct {
	CampaignID      *uint64 `json:"campaign_id,omitempty"`
	CampaignEmailID *uint64 `json:"campaign_email_id,omitempty"`
	ButtonID        *uint64 `json:"button_id,omitempty"`
}

type OnEmailButtonClickResponse struct{}

func (h *campaignHandler) OnEmailButtonClick(ctx context.Context, req *OnEmailButtonClickRequest, res *OnEmailButtonClickResponse) error {
	return nil
}
