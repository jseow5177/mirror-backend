package run_campaigns

import (
	"cdp/config"
	"cdp/dep"
	"cdp/entity"
	"cdp/handler"
	"cdp/pkg/goutil"
	"cdp/pkg/service"
	"cdp/repo"
	"context"
	"fmt"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"math"
	"net/url"
	"time"
)

type RunCampaigns struct {
	cfg            *config.Config
	campaignRepo   repo.CampaignRepo
	emailService   dep.EmailService
	segmentHandler handler.SegmentHandler
	emailHandler   handler.EmailHandler
	tenantRepo     repo.TenantRepo
}

func New(cfg *config.Config, campaignRepo repo.CampaignRepo, emailService dep.EmailService,
	segmentHandler handler.SegmentHandler, emailHandler handler.EmailHandler, tenantRepo repo.TenantRepo) service.Job {
	return &RunCampaigns{
		cfg:            cfg,
		campaignRepo:   campaignRepo,
		emailService:   emailService,
		segmentHandler: segmentHandler,
		emailHandler:   emailHandler,
		tenantRepo:     tenantRepo,
	}
}

func (h *RunCampaigns) Init(_ context.Context) error {
	return nil
}

func (h *RunCampaigns) Run(ctx context.Context) error {
	var (
		taskG   = new(errgroup.Group)
		statusG = new(errgroup.Group)
		c       = 10
		ch      = make(chan struct{}, c)
		now     = time.Now().Unix()
	)

	campaigns, err := h.campaignRepo.GetPendingCampaigns(ctx, uint64(now))
	if err != nil {
		log.Ctx(ctx).Error().Msgf("get campaigns failed: %v", err)
		return err
	}

	log.Ctx(ctx).Info().Msgf("number of campaigns to be processed: %d", len(campaigns))

	type campaignStatus struct {
		err      error
		campaign *entity.Campaign
		status   entity.CampaignStatus
	}

	// track campaign status
	var (
		statusChan           = make(chan campaignStatus, len(campaigns)*100)
		doneChan             = make(chan struct{}, c)
		updateCampaignStatus = func(status entity.CampaignStatus, campaign *entity.Campaign, err error) {
			statusChan <- campaignStatus{err: err, campaign: campaign, status: status}
		}
	)
	statusG.Go(func() error {
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
			case <-doneChan:
				return nil
			}
		}
	})

	for _, campaign := range campaigns {
		select {
		case ch <- struct{}{}:
		}

		campaign := campaign
		taskG.Go(func() error {
			// release go routine
			defer func() {
				<-ch
			}()

			tenant, err := h.tenantRepo.GetByID(ctx, campaign.GetTenantID())
			if err != nil {
				updateCampaignStatus(entity.CampaignStatusFailed, campaign, fmt.Errorf("get tenant failed: %v", err))
				return err
			}

			contextInfo := handler.ContextInfo{
				Tenant: tenant,
			}

			var (
				uds    = make([]*entity.Ud, 0)
				cursor = ""
			)
			for {
				var (
					downloadUdsReq = &handler.DownloadUdsRequest{
						ContextInfo: contextInfo,
						SegmentID:   campaign.SegmentID,
						Pagination: &repo.Pagination{
							Limit:  goutil.Uint32(handler.DefaultMaxLimit),
							Cursor: goutil.String(cursor),
						},
					}
					downloadUdsRes = new(handler.DownloadUdsResponse)
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
			if err := h.campaignRepo.Update(ctx, campaign); err != nil {
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
					getEmailReq = &handler.GetEmailRequest{
						ContextInfo: contextInfo,
						EmailID:     campaignEmail.EmailID,
					}
					getEmailRes = new(handler.GetEmailResponse)
				)
				if err := h.emailHandler.GetEmail(ctx, getEmailReq, getEmailRes); err != nil {
					updateCampaignStatus(entity.CampaignStatusFailed, campaign,
						fmt.Errorf("get email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID()))
					return err
				}

				base64Decoded, err := goutil.Base64Decode(getEmailRes.Email.GetHtml())
				if err != nil {
					updateCampaignStatus(entity.CampaignStatusFailed, campaign,
						fmt.Errorf("decode email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID()))
					return err
				}

				html, err := url.QueryUnescape(base64Decoded)
				if err != nil {
					updateCampaignStatus(entity.CampaignStatusFailed, campaign,
						fmt.Errorf("query unescape email failed: %v, campaign_email_id: %v", err, campaignEmail.GetID()))
					return err
				}

				htmls = append(htmls, html)
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

	taskErr := taskG.Wait()

	doneChan <- struct{}{}

	_ = statusG.Wait()

	return taskErr
}

func (h *RunCampaigns) CleanUp(_ context.Context) error {
	return nil
}
