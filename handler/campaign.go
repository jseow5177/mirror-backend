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
	"strconv"
	"time"
)

type CampaignHandler interface {
	CreateCampaign(ctx context.Context, req *CreateCampaignRequest, res *CreateCampaignResponse) error
	OnEmailAction(ctx context.Context, req *OnEmailActionRequest, res *OnEmailActionResponse) error
	GetCampaigns(ctx context.Context, req *GetCampaignsRequest, res *GetCampaignsResponse) error
	GetCampaign(ctx context.Context, req *GetCampaignRequest, res *GetCampaignResponse) error
}

type campaignHandler struct {
	cfg             *config.Config
	campaignRepo    repo.CampaignRepo
	emailService    dep.EmailService
	segmentHandler  SegmentHandler
	campaignLogRepo repo.CampaignLogRepo
	emailHandler    EmailHandler
}

func NewCampaignHandler(
	cfg *config.Config,
	campaignRepo repo.CampaignRepo,
	emailService dep.EmailService,
	segmentHandler SegmentHandler,
	campaignLogRepo repo.CampaignLogRepo,
	emailHandler EmailHandler,
) CampaignHandler {
	return &campaignHandler{
		cfg,
		campaignRepo,
		emailService,
		segmentHandler,
		campaignLogRepo,
		emailHandler,
	}
}

type GetCampaignsRequest struct {
	ContextInfo
	Keyword    *string          `json:"keyword,omitempty"`
	Pagination *repo.Pagination `json:"pagination,omitempty"`
}

func (req *GetCampaignsRequest) GetKeyword() string {
	if req != nil && req.Keyword != nil {
		return *req.Keyword
	}
	return ""
}

type GetCampaignsResponse struct {
	Campaigns  []*entity.Campaign `json:"campaigns"`
	Pagination *repo.Pagination   `json:"pagination,omitempty"`
}

var GetCampaignsValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo": ContextInfoValidator(true, false),
	"keyword": &validator.String{
		Optional: true,
	},
	"pagination": PaginationValidator(),
})

func (h *campaignHandler) GetCampaigns(ctx context.Context, req *GetCampaignsRequest, res *GetCampaignsResponse) error {
	if err := GetCampaignsValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	if req.Pagination == nil {
		req.Pagination = new(repo.Pagination)
	}

	campaigns, pagination, err := h.campaignRepo.GetManyByKeyword(ctx, req.GetTenantID(), req.GetKeyword(), req.Pagination)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaigns failed: %v", err)
		return err
	}

	res.Campaigns = campaigns
	res.Pagination = pagination

	return nil
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
	ContextInfo
	Name         *string          `json:"name,omitempty"`
	CampaignDesc *string          `json:"campaign_desc,omitempty"`
	SegmentID    *uint64          `json:"segment_id,omitempty"`
	Emails       []*CampaignEmail `json:"emails,omitempty"`
	Schedule     *uint64          `json:"schedule,omitempty"`
}

func (req *CreateCampaignRequest) GetSchedule() uint64 {
	if req != nil && req.Schedule != nil {
		return *req.Schedule
	}
	return 0
}

func (req *CreateCampaignRequest) ToCampaign() *entity.Campaign {
	now := time.Now()

	campaignEmails := make([]*entity.CampaignEmail, 0, len(req.Emails))
	for _, campaignEmail := range req.Emails {
		campaignEmails = append(campaignEmails, &entity.CampaignEmail{
			EmailID: campaignEmail.EmailID,
			Subject: campaignEmail.Subject,
			Ratio:   campaignEmail.Ratio,
		})
	}

	return &entity.Campaign{
		Name:           req.Name,
		CampaignDesc:   req.CampaignDesc,
		SegmentID:      req.SegmentID,
		SegmentSize:    goutil.Uint64(0),
		Progress:       goutil.Uint64(0),
		CampaignEmails: campaignEmails,
		Status:         entity.CampaignStatusPending,
		CreatorID:      goutil.Uint64(req.GetUserID()),
		TenantID:       goutil.Uint64(req.GetTenantID()),
		Schedule:       goutil.Uint64(req.GetSchedule()),
		CreateTime:     goutil.Uint64(uint64(now.Unix())),
		UpdateTime:     goutil.Uint64(uint64(now.Unix())),
	}
}

type CreateCampaignResponse struct {
	Campaign *entity.Campaign `json:"campaign"`
}

var CreateCampaignValidator = validator.MustForm(map[string]validator.Validator{
	"ContextInfo":   ContextInfoValidator(false, false),
	"name":          ResourceNameValidator(false),
	"campaign_desc": ResourceDescValidator(false),
	"segment_id":    &validator.UInt64{},
	"emails": &validator.Slice{
		MinLen: 1,
		MaxLen: 4,
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
		return errutil.ValidationError(err)
	}

	// validate ratio
	var ratio uint64
	for _, email := range req.Emails {
		ratio += email.GetRatio()
	}
	if ratio != 100 {
		return errutil.ValidationError(errors.New("ratios must add up to 100"))
	}

	campaign := req.ToCampaign()

	for _, campaignEmail := range campaign.CampaignEmails {
		var (
			getEmailReq = &GetEmailRequest{
				ContextInfo: req.ContextInfo,
				EmailID:     campaignEmail.EmailID,
			}
			getEmailRes = new(GetEmailResponse)
		)
		if err := h.emailHandler.GetEmail(ctx, getEmailReq, getEmailRes); err != nil {
			log.Ctx(ctx).Error().Msgf("get email err: %v", err)
			return err
		}
	}

	id, err := h.campaignRepo.Create(ctx, campaign)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign failed: %v", err)
		return err
	}

	campaign.ID = goutil.Uint64(id)
	res.Campaign = campaign

	return nil
}

type OnEmailActionRequest struct {
	Event   *string  `json:"event,omitempty"`
	Link    *string  `json:"link,omitempty"`
	TsEvent *uint64  `json:"ts_event,omitempty"`
	Email   *string  `json:"email,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

func (r *OnEmailActionRequest) GetEvent() string {
	if r != nil && r.Event != nil {
		return *r.Event
	}
	return ""
}

type OnEmailActionResponse struct{}

func (h *campaignHandler) OnEmailAction(ctx context.Context, req *OnEmailActionRequest, _ *OnEmailActionResponse) error {
	event, ok := entity.SupportedEvents[req.GetEvent()]
	if !ok {
		log.Ctx(ctx).Debug().Msgf("unsupported event: %v", req.GetEvent())
		return nil
	}

	if len(req.Tags) == 0 {
		log.Ctx(ctx).Warn().Msg("empty tags, expect one campaign_email_id")
		return nil
	}

	campaignEmailID, err := strconv.ParseUint(req.Tags[0], 10, 64)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("parse campaign_email_id err: %v", err)
		return err
	}

	campaignLog := &entity.CampaignLog{
		CampaignEmailID: goutil.Uint64(campaignEmailID),
		Event:           event,
		Link:            req.Link,
		Email:           req.Email,
		EventTime:       req.TsEvent,
		CreateTime:      goutil.Uint64(uint64(time.Now().Unix())),
	}

	if err := h.campaignLogRepo.CreateMany(ctx, []*entity.CampaignLog{campaignLog}); err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign log failed: %v", err)
		return err
	}

	return nil
}

type GetCampaignRequest struct {
	ContextInfo
	CampaignID *uint64 `json:"campaign_id,omitempty"`
}

func (r *GetCampaignRequest) GetCampaignID() uint64 {
	if r != nil && r.CampaignID != nil {
		return *r.CampaignID
	}
	return 0
}

type GetCampaignResponse struct {
	Campaign *entity.Campaign `json:"campaign,omitempty"`
	Segment  *entity.Segment  `json:"segment,omitempty"`
}

var GetCampaignValidator = validator.MustForm(map[string]validator.Validator{
	"campaign_id": &validator.UInt64{},
})

func (h *campaignHandler) GetCampaign(ctx context.Context, req *GetCampaignRequest, res *GetCampaignResponse) error {
	if err := GetCampaignValidator.Validate(req); err != nil {
		return errutil.ValidationError(err)
	}

	campaign, err := h.campaignRepo.GetByID(ctx, req.GetTenantID(), req.GetCampaignID())
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaign err: %v", err)
		return err
	}

	var (
		getSegmentReq = &GetSegmentRequest{
			ContextInfo: req.ContextInfo,
			SegmentID:   campaign.SegmentID,
		}
		getSegmentRes = new(GetSegmentResponse)
	)
	if err := h.segmentHandler.GetSegment(ctx, getSegmentReq, getSegmentRes); err != nil {
		log.Ctx(ctx).Error().Msgf("get segment err: %v", err)
		return err
	}

	for _, campaignEmail := range campaign.CampaignEmails {
		campaignResult := new(entity.CampaignResult)

		var (
			getEmailReq = &GetEmailRequest{
				ContextInfo: req.ContextInfo,
				EmailID:     campaignEmail.EmailID,
			}
			getEmailRes = new(GetEmailResponse)
		)
		if err := h.emailHandler.GetEmail(ctx, getEmailReq, getEmailRes); err != nil {
			log.Ctx(ctx).Error().Msgf("get email err: %v", err)
			return err
		}
		email := getEmailRes.Email
		campaignEmail.Email = email

		totalUniqueOpen, err := h.campaignLogRepo.CountTotalUniqueOpen(ctx, campaignEmail.GetID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("count campaign email total unique open err: %v, campaign_email_id: %v", err, campaignEmail.GetID())
			return err
		}

		clickCountsByLink, err := h.campaignLogRepo.CountClicksByLink(ctx, campaignEmail.GetID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("count campaign email clicks err: %v, campaign_email_id: %v", err, campaignEmail.GetID())
			return err
		}

		var totalClicks uint64
		for _, clickCount := range clickCountsByLink {
			totalClicks += clickCount
		}

		avgOpenTime, err := h.campaignLogRepo.GetAvgOpenTime(ctx, campaignEmail.GetID())
		if err != nil {
			log.Ctx(ctx).Error().Msgf("get email avg open time err: %v, campaign_email_id: %v", err, campaignEmail.GetID())
			return err
		}

		campaignResult.TotalUniqueOpenCount = goutil.Uint64(totalUniqueOpen)
		campaignResult.TotalClickCount = goutil.Uint64(totalClicks)
		campaignResult.ClickCountsByLink = clickCountsByLink
		campaignResult.AvgOpenTime = goutil.Uint64(avgOpenTime)

		campaignEmail.CampaignResult = campaignResult
	}

	res.Campaign = campaign
	res.Segment = getSegmentRes.Segment

	return nil
}
