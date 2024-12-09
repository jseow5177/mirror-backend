package handler

import (
	"bytes"
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
	"fmt"
	"github.com/dchest/uniuri"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
	"math"
	"strings"
	"time"
)

type CampaignHandler interface {
	CreateCampaign(ctx context.Context, req *CreateCampaignRequest, res *CreateCampaignResponse) error
	RunCampaigns(ctx context.Context, req *RunCampaignsRequest, res *RunCampaignsResponse) error
	OnEmailOpen(ctx context.Context, req *OnEmailOpenRequest, res *OnEmailOpenResponse) error
	OnEmailButtonClick(ctx context.Context, req *OnEmailButtonClickRequest, res *OnEmailButtonClickResponse) error
	OnEmailAction(_ context.Context, _ *OnEmailActionRequest, _ *OnEmailActionResponse) error
}

type campaignHandler struct {
	cfg            *config.Config
	campaignRepo   repo.CampaignRepo
	emailRepo      repo.EmailRepo
	emailService   dep.EmailService
	segmentHandler SegmentHandler
}

func NewCampaignHandler(
	cfg *config.Config,
	campaignRepo repo.CampaignRepo,
	emailRepo repo.EmailRepo,
	emailService dep.EmailService,
	segmentHandler SegmentHandler,
) CampaignHandler {
	return &campaignHandler{
		cfg,
		campaignRepo,
		emailRepo,
		emailService,
		segmentHandler,
	}
}

type RunCampaignsRequest struct{}

type RunCampaignsResponse struct{}

func (h *campaignHandler) RunCampaigns(ctx context.Context, req *RunCampaignsRequest, res *RunCampaignsResponse) error {
	go func() {
		var (
			ctx = context.WithoutCancel(ctx)
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
			return
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

				defer func() {
					if err != nil {
						if err = h.campaignRepo.Update(ctx, f, &entity.Campaign{
							Status: entity.CampaignStatusFailed,
						}); err != nil {
							log.Ctx(ctx).Error().Msgf("set campaign to failed err: %v, campaign_id: %v", err, campaign)
						}
					}
					<-ch
				}()

				res := new(GetUdsResponse)
				if err = h.segmentHandler.GetUds(ctx, &GetUdsRequest{
					SegmentID: campaign.SegmentID,
				}, res); err != nil {
					log.Ctx(ctx).Error().Msgf("get uds failed: %v, campaign_id: %v", err, campaign)
					return err
				}

				if err = h.campaignRepo.Update(ctx, f, &entity.Campaign{
					Status: entity.CampaignStatusRunning,
				}); err != nil {
					log.Ctx(ctx).Error().Msgf("set campaign to running err: %v, campaign_id: %v", err, campaign)
					return err
				}

				var (
					pos            int
					campaignEmails = campaign.CampaignEmails
					emailBuckets   = make([][]string, 0)
				)
				for _, campaignEmail := range campaignEmails {
					count := int(math.Ceil(float64(len(res.Uds)) * float64((campaignEmail.GetRatio())/100)))
					end := int(math.Min(float64(len(res.Uds)), float64(pos+count)))

					emailBucket := make([]string, 0)
					for _, ud := range res.Uds[pos:end] {
						emailBucket = append(emailBucket, ud.GetID())
					}

					emailBuckets = append(emailBuckets, emailBucket)

					pos += count
				}

				for i, emailBucket := range emailBuckets {
					campaignEmail := campaignEmails[i]

					var decodedHtml []byte
					decodedHtml, err = base64.StdEncoding.DecodeString(campaignEmail.GetHtml())
					if err != nil {
						log.Ctx(ctx).Error().Msgf("decode email html failed: %v, campaign_email_id: %v", err, campaignEmail.GetID())
						return err
					}

					to := make([]dep.Receiver, 0)
					for _, email := range emailBucket {
						to = append(to, dep.Receiver{
							Email: email,
						})
					}

					sendSmtpEmail := dep.SendSmtpEmail{
						CampaignEmailID: campaignEmail.GetID(),
						From: dep.Sender{
							Email: "mirrorcdp@gmail.com",
						},
						To:          to,
						Subject:     campaignEmail.GetSubject(),
						HtmlContent: string(decodedHtml),
					}

					if err = h.emailService.SendEmail(ctx, sendSmtpEmail); err != nil {
						log.Ctx(ctx).Error().Msgf("send email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID())
						return err
					}
				}

				return nil
			})
		}
	}()

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
			EmailID:     email.EmailID,
			Subject:     email.Subject,
			Html:        goutil.String(""),
			Ratio:       email.Ratio,
			OpenCount:   goutil.Uint64(uint64(0)),
			ClickCounts: make(map[string]uint64),
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

	for i, emailHtml := range emailHtmls {
		campaignEmailID := campaignEmails[i].GetID()

		decodedHtml, err := base64.StdEncoding.DecodeString(emailHtml)
		if err != nil {
			log.Ctx(ctx).Error().Msgf("decode email html failed: %v, campaign_email_id: %v", err, campaignEmailID)
			return err
		}

		doc, err := html.Parse(strings.NewReader(string(decodedHtml)))
		if err != nil {
			log.Ctx(ctx).Error().Msgf("parse email html failed: %v, campaign_email_id: %v", err, campaignEmailID)
			return err
		}

		h.addEmailOpenTracker(doc, campaignEmailID)
		buttonIDs := h.addButtonClickTracker(doc, campaignEmailID)

		var b bytes.Buffer
		if err := html.Render(&b, doc); err != nil {
			log.Ctx(ctx).Error().Msgf("render email html failed: %v, campaign_email_id: %v", err, campaignEmailID)
			return err
		}

		f := &repo.CampaignEmailFilter{
			ID: goutil.Uint64(campaignEmailID),
		}

		clickCounts := make(map[string]uint64)
		for _, buttonID := range buttonIDs {
			clickCounts[buttonID] = 0
		}
		encodedHtml := base64.StdEncoding.EncodeToString(b.Bytes())

		campaignEmails[i].ClickCounts = clickCounts
		campaignEmails[i].Html = goutil.String(encodedHtml)

		if err := h.campaignRepo.UpdateCampaignEmail(ctx, f, campaignEmails[i]); err != nil {
			log.Ctx(ctx).Error().Msgf("update campaign email html err: %v, campaign_email_id: %v", err, campaignEmailID)
			return err
		}
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

func (h *campaignHandler) OnEmailOpen(ctx context.Context, req *OnEmailOpenRequest, _ *OnEmailOpenResponse) error {
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
	campaignEmail.OpenCount = goutil.Uint64(campaignEmail.GetOpenCount() + 1)

	if err := h.campaignRepo.UpdateCampaignEmail(ctx, f, campaignEmail); err != nil {
		log.Ctx(ctx).Error().Msgf("update campaign email err: %v, campaign_email_id: %v", err, req.CampaignEmailID)
		return err
	}

	return nil
}

type OnEmailButtonClickRequest struct {
	CampaignEmailID *uint64 `schema:"campaign_email_id,omitempty"`
	ButtonID        *string `schema:"button_id,omitempty"`
}

func (r *OnEmailButtonClickRequest) GetButtonID() string {
	if r != nil && r.ButtonID != nil {
		return *r.ButtonID
	}
	return ""
}

type OnEmailButtonClickResponse struct{}

var OnEmailButtonClickValidator = validator.MustForm(map[string]validator.Validator{
	"campaign_email_id": &validator.UInt64{},
	"button_id":         &validator.String{},
})

func (h *campaignHandler) OnEmailButtonClick(ctx context.Context, req *OnEmailButtonClickRequest, _ *OnEmailButtonClickResponse) error {
	if err := OnEmailButtonClickValidator.Validate(req); err != nil {
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

	clickCounts := campaignEmail.GetClickCounts()
	clickCounts[req.GetButtonID()]++
	campaignEmail.ClickCounts = clickCounts

	if err := h.campaignRepo.UpdateCampaignEmail(ctx, f, campaignEmail); err != nil {
		log.Ctx(ctx).Error().Msgf("update campaign email err: %v, campaign_email_id: %v", err, req.CampaignEmailID)
		return err
	}

	return nil
}

func (h *campaignHandler) addEmailOpenTracker(doc *html.Node, campaignEmailID uint64) {
	var body *html.Node

	// find <body> tag
	var findBody func(*html.Node)
	findBody = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "body" {
			body = n
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findBody(c)
		}
	}
	findBody(doc)

	if body != nil {
		// create <img> tag
		body.AppendChild(&html.Node{
			Type: html.ElementNode,
			Data: "img",
			Attr: []html.Attribute{
				{Key: "src", Val: fmt.Sprintf("%s/api/v1/on_email_button_click?campaign_email_id=%d",
					h.cfg.ServerDomain, campaignEmailID)},
				{Key: "alt", Val: ""},
				{Key: "src", Val: "display:none;"},
			},
		})
	}
}

func (h *campaignHandler) addButtonClickTracker(doc *html.Node, campaignEmailID uint64) []string {
	buttonIDs := make([]string, 0)

	var add func(*html.Node)
	add = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			buttonID := uniuri.New()

			buttonIDs = append(buttonIDs, buttonID)

			n.Attr = append(n.Attr, html.Attribute{
				Key: "id",
				Val: buttonID,
			})

			n.Attr = append(n.Attr, html.Attribute{
				Key: "onclick",
				Val: fmt.Sprintf("fetch('%s/api/v1/on_email_button_click?campaign_email_id=%d&button_id=%s')",
					h.cfg.ServerDomain, campaignEmailID, buttonID),
			})
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			add(c)
		}
	}
	add(doc)

	return buttonIDs
}

type OnEmailActionRequest struct{}

type OnEmailActionResponse struct{}

func (h *campaignHandler) OnEmailAction(_ context.Context, _ *OnEmailActionRequest, _ *OnEmailActionResponse) error {
	fmt.Println("action!")
	return nil
}
