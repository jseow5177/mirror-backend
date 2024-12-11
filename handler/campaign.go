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
	"encoding/base64"
	"errors"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"math"
	"strconv"
	"time"
)

type CampaignHandler interface {
	CreateCampaign(ctx context.Context, req *CreateCampaignRequest, res *CreateCampaignResponse) error
	RunCampaigns(ctx context.Context, req *RunCampaignsRequest, res *RunCampaignsResponse) error
	OnEmailAction(_ context.Context, _ *OnEmailActionRequest, _ *OnEmailActionResponse) error
}

type campaignHandler struct {
	cfg             *config.Config
	campaignRepo    repo.CampaignRepo
	emailRepo       repo.EmailRepo
	emailService    dep.EmailService
	segmentHandler  SegmentHandler
	campaignLogRepo repo.CampaignLogRepo
}

func NewCampaignHandler(
	cfg *config.Config,
	campaignRepo repo.CampaignRepo,
	emailRepo repo.EmailRepo,
	emailService dep.EmailService,
	segmentHandler SegmentHandler,
	campaignLogRepo repo.CampaignLogRepo,
) CampaignHandler {
	return &campaignHandler{
		cfg,
		campaignRepo,
		emailRepo,
		emailService,
		segmentHandler,
		campaignLogRepo,
	}
}

type RunCampaignsRequest struct{}

type RunCampaignsResponse struct{}

func (h *campaignHandler) RunCampaigns(ctx context.Context, req *RunCampaignsRequest, res *RunCampaignsResponse) error {
	ctx = context.WithoutCancel(ctx)

	var (
		g   = new(errgroup.Group)
		c   = 10
		ch  = make(chan struct{}, c)
		now = time.Now().Unix()
	)

	// fetch without limit
	campaigns, _, err := h.campaignRepo.GetMany(ctx, &repo.CampaignFilter{
		Conditions: []*repo.Condition{
			{Field: "status", Op: repo.OpEq, Value: entity.CampaignStatusPending, NextLogicalOp: repo.And},
			{Field: "schedule", Op: repo.OpLte, Value: now, NextLogicalOp: repo.And},
		},
		Pagination: new(repo.Pagination),
	})
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaigns failed: %v", err)
		return err
	}

	for _, campaign := range campaigns {
		select {
		case ch <- struct{}{}:
		}

		campaign := campaign
		g.Go(func() error {
			var err error

			f := &repo.CampaignFilter{
				Conditions: []*repo.Condition{
					{Field: "id", Op: repo.OpEq, Value: campaign.GetID()},
				},
			}

			// release go routine
			defer func() {
				<-ch
			}()

			// set campaign status
			defer func() {
				if err != nil {
					if err = h.campaignRepo.Update(ctx, f, &entity.Campaign{
						Status: entity.CampaignStatusFailed,
					}); err != nil {
						log.Ctx(ctx).Error().Msgf("set campaign to failed err: %v, campaign_id: %v", err, campaign.GetID())
					}
				}
			}()

			// fetch segment users
			getUdsRes := new(GetUdsResponse)
			if err = h.segmentHandler.GetUds(ctx, &GetUdsRequest{
				SegmentID: campaign.SegmentID,
			}, getUdsRes); err != nil {
				log.Ctx(ctx).Error().Msgf("get uds failed: %v, campaign_id: %v", err, campaign.GetID())
				return err
			}

			// set campaign to Running, update the segment size
			if err = h.campaignRepo.Update(ctx, f, &entity.Campaign{
				SegmentSize: goutil.Uint64(uint64(len(getUdsRes.Uds))),
				Status:      entity.CampaignStatusRunning,
			}); err != nil {
				log.Ctx(ctx).Error().Msgf("set campaign to running err: %v, campaign_id: %v", err, campaign.GetID())
				return err
			}

			// group emails into buckets and fetch htmls
			var (
				pos            int
				htmls          = make([]string, 0)
				campaignEmails = campaign.CampaignEmails
				emailBuckets   = make([][]string, 0)
			)
			for _, campaignEmail := range campaignEmails {
				// group emails
				count := int(math.Ceil(float64(len(getUdsRes.Uds)) * float64((campaignEmail.GetRatio())/100)))
				end := int(math.Min(float64(len(getUdsRes.Uds)), float64(pos+count)))

				emailBucket := make([]string, 0)
				for _, ud := range getUdsRes.Uds[pos:end] {
					emailBucket = append(emailBucket, ud.GetID())
				}

				emailBuckets = append(emailBuckets, emailBucket)

				pos += count

				// fetch htmls
				var email *entity.Email
				email, err = h.emailRepo.Get(ctx, &repo.EmailFilter{
					ID: campaignEmail.EmailID,
				})
				if err != nil {
					log.Ctx(ctx).Error().Msgf("get email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID())
					return err
				}

				var decodedHtml []byte
				decodedHtml, err = base64.StdEncoding.DecodeString(email.GetHtml())
				if err != nil {
					log.Ctx(ctx).Error().Msgf("decode email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID())
					return err
				}
				htmls = append(htmls, string(decodedHtml))
			}

			// send out emails by buckets
			for i, emailBucket := range emailBuckets {
				var (
					progress      int
					campaignEmail = campaignEmails[i]
					batchSize     = dep.MaxRecipientsPerSend
				)
				for start := 0; start < len(emailBucket); start += batchSize {
					end := start + batchSize
					if end > len(emailBucket) {
						end = len(emailBucket)
					}

					// Create a batch of recipients
					to := make([]dep.Receiver, 0, end-start)
					for _, email := range emailBucket[start:end] {
						to = append(to, dep.Receiver{
							Email: email,
						})
					}

					progress += len(to)

					sendSmtpEmail := dep.SendSmtpEmail{
						CampaignEmailID: campaignEmail.GetID(),
						From: dep.Sender{
							Email: "mirrorcdp@gmail.com",
						},
						To:          to,
						Subject:     campaignEmail.GetSubject(),
						HtmlContent: htmls[i],
					}

					// Send the email and handle errors
					if err = h.emailService.SendEmail(ctx, sendSmtpEmail); err != nil {
						log.Ctx(ctx).Error().Msgf("send email failed: %v, campaign_email_id: %v, batch_start: %d, batch_end: %d", err,
							campaignEmail.GetID(), start, end)
					} else {
						if err = h.campaignRepo.Update(ctx, f, &entity.Campaign{
							Progress: goutil.Uint64(uint64(progress)),
						}); err != nil {
							log.Ctx(ctx).Error().Msgf("update campaign progress err: %v, campaign_id: %v, progress: %v", err,
								campaign.GetID(), progress)
						}
					}

					time.Sleep(1 * time.Second)
				}
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

	emailHtmls := make([]string, 0)
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
			EmailID: email.EmailID,
			Subject: email.Subject,
			Ratio:   email.Ratio,
		})

		emailHtmls = append(emailHtmls, e.GetHtml())
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

type OnEmailActionRequest struct {
	Event   *string  `json:"event,omitempty"`
	Link    *string  `json:"link,omitempty"`
	TsEpoch *uint64  `json:"ts_epoch,omitempty"`
	Date    *string  `json:"date,omitempty"`
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
		LogExtra: entity.LogExtra{
			Link:    req.Link,
			Email:   req.Email,
			Date:    req.Date,
			TsEpoch: req.TsEpoch,
		},
		CreateTime: goutil.Uint64(uint64(time.Now().Unix())),
	}

	if err := h.campaignLogRepo.BatchCreate(ctx, []*entity.CampaignLog{campaignLog}); err != nil {
		log.Ctx(ctx).Error().Msgf("create campaign log failed: %v", err)
		return err
	}

	return nil
}
