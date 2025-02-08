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
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"math"
	"strconv"
	"time"
)

type CampaignHandler interface {
	CreateCampaign(ctx context.Context, req *CreateCampaignRequest, res *CreateCampaignResponse) error
	RunCampaigns(ctx context.Context, req *RunCampaignsRequest, res *RunCampaignsResponse) error
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

func (r *GetCampaignsRequest) GetKeyword() string {
	if r != nil && r.Keyword != nil {
		return *r.Keyword
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

	campaigns, pagination, err := h.campaignRepo.GetByKeyword(ctx, req.GetTenantID(), req.GetKeyword(), req.Pagination)
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaigns failed: %v", err)
		return err
	}

	res.Campaigns = campaigns
	res.Pagination = pagination

	return nil
}

type RunCampaignsRequest struct{}

type RunCampaignsResponse struct{}

func (h *campaignHandler) RunCampaigns(ctx context.Context, _ *RunCampaignsRequest, _ *RunCampaignsResponse) error {
	ctx = context.WithoutCancel(ctx)

	var (
		g   = new(errgroup.Group)
		c   = 10
		ch  = make(chan struct{}, c)
		now = time.Now().Unix()
	)

	campaigns, err := h.campaignRepo.GetPendingCampaigns(ctx, uint64(now))
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaigns failed: %v", err)
		return err
	}

	type campaignStatus struct {
		err      error
		campaign *entity.Campaign
		status   entity.CampaignStatus
	}

	// track campaign status
	var (
		statusChan           = make(chan campaignStatus, len(campaigns))
		updateCampaignStatus = func(status entity.CampaignStatus, campaign *entity.Campaign, err error) {
			statusChan <- campaignStatus{err: err, campaign: campaign, status: status}
		}
	)
	g.Go(func() error {
		for {
			select {
			case ce := <-statusChan:
				campaign := ce.campaign
				if ce.err != nil {
					log.Ctx(ctx).Error().Msgf("[campaign ID %d] error encountered: %v", campaign.GetID(), ce.err)
				}

				campaign.Update(&entity.Campaign{
					Status: ce.status,
				})
				if err = h.campaignRepo.Update(ctx, campaign); err != nil {
					log.Ctx(ctx).Error().Msgf("[campaign ID %d] set campaign status failed: %v, status: %v", campaign.GetID(), err, ce.status)
				}
			}
		}
	})

	for _, campaign := range campaigns {
		select {
		case ch <- struct{}{}:
		}

		campaign := campaign
		g.Go(func() error {
			// release go routine
			defer func() {
				<-ch
			}()

			contextInfo := ContextInfo{
				Tenant: &entity.Tenant{
					ID: campaign.TenantID,
				},
			}

			var (
				uds    = make([]*entity.Ud, 0)
				cursor = ""
			)
			for {
				var (
					downloadUdsReq = &DownloadUdsRequest{
						ContextInfo: contextInfo,
						SegmentID:   campaign.SegmentID,
						Pagination: &repo.Pagination{
							Limit:  goutil.Uint32(DefaultMaxLimit),
							Cursor: goutil.String(cursor),
						},
					}
					downloadUdsRes = new(DownloadUdsResponse)
				)

				if err := h.segmentHandler.DownloadUds(ctx, downloadUdsReq, downloadUdsRes); err != nil {
					updateCampaignStatus(entity.CampaignStatusFailed, campaign, fmt.Errorf("download uds failed: %v", err))
					return err
				}

				cursor = downloadUdsRes.GetPagination().GetCursor()
				uds = append(uds, downloadUdsRes.Uds...)

				if cursor == "" {
					break
				}
			}

			// set campaign to Running, update the segment size
			campaign.Update(&entity.Campaign{
				SegmentSize: goutil.Uint64(uint64(len(uds))),
				Status:      entity.CampaignStatusRunning,
			})
			if err = h.campaignRepo.Update(ctx, campaign); err != nil {
				updateCampaignStatus(entity.CampaignStatusFailed, campaign, fmt.Errorf("set campaign to running failed: %v", err))
				return err
			}

			// group emails into buckets and fetch htmls
			var (
				pos            int
				htmls          = make([]string, 0)
				emailBuckets   = make([][]string, 0)
				campaignEmails = campaign.CampaignEmails
			)
			for _, campaignEmail := range campaignEmails {
				// group emails
				var (
					count = int(math.Ceil(float64(len(uds)) * float64(campaignEmail.GetRatio()) / float64(100)))
					end   = int(math.Min(float64(len(uds)), float64(pos+count)))
				)

				emailBucket := make([]string, 0)
				for _, ud := range uds[pos:end] {
					emailBucket = append(emailBucket, ud.GetID())
				}

				emailBuckets = append(emailBuckets, emailBucket)

				pos += count

				// fetch HTMLs
				var (
					getEmailReq = &GetEmailRequest{
						ContextInfo: contextInfo,
						EmailID:     campaignEmail.EmailID,
					}
					getEmailRes = new(GetEmailResponse)
				)
				if err := h.emailHandler.GetEmail(ctx, getEmailReq, getEmailRes); err != nil {
					updateCampaignStatus(entity.CampaignStatusFailed, campaign,
						fmt.Errorf("get email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID()))
					return err
				}

				var decodedHtml string
				decodedHtml, err = goutil.Base64Decode(getEmailRes.Email.GetHtml())
				if err != nil {
					updateCampaignStatus(entity.CampaignStatusFailed, campaign,
						fmt.Errorf("decode email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID()))
					return err
				}
				htmls = append(htmls, decodedHtml)
			}

			// send out emails by buckets
			for i, emailBucket := range emailBuckets {
				var (
					count         uint64
					campaignEmail = campaignEmails[i]
					batchSize     = dep.MaxRecipientsPerSend
				)
				for start := 0; start < len(emailBucket); start += batchSize {
					end := start + batchSize
					if end > len(emailBucket) {
						end = len(emailBucket)
					}

					// Create a batch of recipients
					to := make([]*dep.Receiver, 0, end-start)
					for _, email := range emailBucket[start:end] {
						to = append(to, &dep.Receiver{
							Email: email,
						})
					}

					count += uint64(len(to))

					sendSmtpEmail := &dep.SendSmtpEmail{
						CampaignEmailID: campaignEmail.GetID(),
						From: &dep.Sender{
							Email: "mirrorcdp@gmail.com", // TODO: Set to tenant's email
						},
						To:          to,
						Subject:     campaignEmail.GetSubject(),
						HtmlContent: htmls[i],
					}

					var progress uint64
					if campaign.GetSegmentSize() > 0 {
						progress = count * 100 / campaign.GetSegmentSize()
					}

					// Send emails
					// Log error only, keep the campaign going
					if err := h.emailService.SendEmail(ctx, sendSmtpEmail); err != nil {
						updateCampaignStatus(entity.CampaignStatusRunning, campaign,
							fmt.Errorf("send email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID()))
					}

					// Update progress
					// Log error only, keep the campaign going
					campaign.Update(&entity.Campaign{
						Progress: goutil.Uint64(progress),
					})
					if err := h.campaignRepo.Update(ctx, campaign); err != nil {
						updateCampaignStatus(entity.CampaignStatusRunning, campaign,
							fmt.Errorf("update campaign progress failed: %v, campaign_email_id: %v, progress: %v", err, campaignEmail.GetID(), progress))
					}
				}
			}

			// Send emails done
			campaign.Update(&entity.Campaign{
				Progress: goutil.Uint64(100),
			})
			if err := h.campaignRepo.Update(ctx, campaign); err != nil {
				updateCampaignStatus(entity.CampaignStatusFailed, campaign, fmt.Errorf("set campaign to 100%% completion failed: %v", err))
				return err
			}

			return nil
		})
	}

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

func (r *CreateCampaignRequest) GetSchedule() uint64 {
	if r != nil && r.Schedule != nil {
		return *r.Schedule
	}
	return 0
}

func (r *CreateCampaignRequest) ToCampaign() *entity.Campaign {
	now := time.Now()

	campaignEmails := make([]*entity.CampaignEmail, 0, len(r.Emails))
	for _, campaignEmail := range r.Emails {
		campaignEmails = append(campaignEmails, &entity.CampaignEmail{
			EmailID: campaignEmail.EmailID,
			Subject: campaignEmail.Subject,
			Ratio:   campaignEmail.Ratio,
		})
	}

	return &entity.Campaign{
		Name:           r.Name,
		CampaignDesc:   r.CampaignDesc,
		SegmentID:      r.SegmentID,
		SegmentSize:    goutil.Uint64(0),
		Progress:       goutil.Uint64(0),
		CampaignEmails: campaignEmails,
		Status:         entity.CampaignStatusPending,
		CreatorID:      goutil.Uint64(r.GetUserID()),
		TenantID:       goutil.Uint64(r.GetTenantID()),
		Schedule:       goutil.Uint64(r.GetSchedule()),
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
