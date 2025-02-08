package repo

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"fmt"
	"gorm.io/gorm"
)

var (
	ErrCampaignNotFound = errutil.NotFoundError(errors.New("campaign not found"))
)

type CampaignEmail struct {
	ID         *uint64
	CampaignID *uint64
	EmailID    *uint64
	Subject    *string
	Ratio      *uint64
}

func (m *CampaignEmail) TableName() string {
	return "campaign_email_tab"
}

type Campaign struct {
	ID           *uint64
	Name         *string
	CampaignDesc *string
	SegmentID    *uint64
	SegmentSize  *uint64
	Schedule     *uint64
	Progress     *uint64
	Status       *uint32
	CreatorID    *uint64
	TenantID     *uint64
	CreateTime   *uint64
	UpdateTime   *uint64
}

func (m *Campaign) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

func (m *Campaign) TableName() string {
	return "campaign_tab"
}

func (m *Campaign) GetID() uint64 {
	if m != nil && m.ID != nil {
		return *m.ID
	}
	return 0
}

type CampaignRepo interface {
	Create(ctx context.Context, campaign *entity.Campaign) (uint64, error)
	GetByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Campaign, *Pagination, error)
	GetPendingCampaigns(ctx context.Context, schedule uint64) ([]*entity.Campaign, error)
	GetByID(ctx context.Context, tenantID, campaignID uint64) (*entity.Campaign, error)
	Update(ctx context.Context, tenant *entity.Campaign) error
}

type campaignRepo struct {
	baseRepo BaseRepo
}

func NewCampaignRepo(_ context.Context, baseRepo BaseRepo) CampaignRepo {
	return &campaignRepo{baseRepo: baseRepo}
}

func (r *campaignRepo) GetByID(ctx context.Context, tenantID, campaignID uint64) (*entity.Campaign, error) {
	return r.get(ctx, tenantID, []*Condition{
		{
			Field: "id",
			Value: campaignID,
			Op:    OpEq,
		},
	}, true)
}

func (r *campaignRepo) Update(ctx context.Context, campaign *entity.Campaign) error {
	if err := r.baseRepo.Update(ctx, ToCampaignModel(campaign)); err != nil {
		return err
	}

	return nil
}

func (r *campaignRepo) Create(ctx context.Context, campaign *entity.Campaign) (uint64, error) {
	campaignModel := ToCampaignModel(campaign)

	if err := r.baseRepo.RunTx(ctx, func(ctx context.Context) error {
		if err := r.baseRepo.Create(ctx, campaignModel); err != nil {
			return err
		}

		var (
			campaignEmails      = campaign.CampaignEmails
			campaignEmailModels = make([]*CampaignEmail, len(campaignEmails))
		)
		for i, campaignEmail := range campaign.CampaignEmails {
			campaignEmailModels[i] = ToCampaignEmailModel(campaignModel.GetID(), campaignEmail)
		}

		if err := r.baseRepo.CreateMany(ctx, new(CampaignEmail), campaignEmailModels); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return campaignModel.GetID(), nil
}

func (r *campaignRepo) GetPendingCampaigns(ctx context.Context, schedule uint64) ([]*entity.Campaign, error) {
	campaigns, _, err := r.getMany(ctx, 0, []*Condition{
		{
			Field:         "status",
			Op:            OpEq,
			Value:         entity.CampaignStatusPending,
			NextLogicalOp: LogicalOpAnd,
		},
		{
			Field: "schedule",
			Op:    OpLte,
			Value: schedule,
		},
	}, false, nil)
	if err != nil {
		return nil, err
	}
	return campaigns, nil
}

func (r *campaignRepo) GetByKeyword(ctx context.Context, tenantID uint64, keyword string, p *Pagination) ([]*entity.Campaign, *Pagination, error) {
	return r.getMany(ctx, tenantID, []*Condition{
		{
			Field:         "LOWER(name)",
			Value:         fmt.Sprintf("%%%s%%", keyword),
			Op:            OpLike,
			NextLogicalOp: LogicalOpOr,
			OpenBracket:   true,
		},
		{
			Field:        "LOWER(campaign_desc)",
			Value:        fmt.Sprintf("%%%s%%", keyword),
			Op:           OpLike,
			CloseBracket: true,
		},
	}, true, p)
}

func (r *campaignRepo) getMany(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool, p *Pagination) ([]*entity.Campaign, *Pagination, error) {
	baseConditions := make([]*Condition, 0)
	if tenantID != 0 {
		baseConditions = append(baseConditions, r.getBaseConditions(tenantID)...)
	}

	res, pNew, err := r.baseRepo.GetMany(ctx, new(Campaign), &Filter{
		Conditions: append(baseConditions, r.mayAddDeleteFilter(conditions, filterDelete)...),
		Pagination: p,
	})
	if err != nil {
		return nil, nil, err
	}

	var (
		campaigns   = make([]*entity.Campaign, 0, len(res))
		campaignIDs = make([]uint64, 0, len(res))
	)
	for _, m := range res {
		campaign := ToCampaign(m.(*Campaign))
		campaigns = append(campaigns, campaign)
		campaignIDs = append(campaignIDs, campaign.GetID())
	}

	campaignEmails, err := r.getCampaignEmails(ctx, campaignIDs...)
	if err != nil {
		return nil, nil, err
	}

	campaignIDsToEmails := make(map[uint64][]*entity.CampaignEmail)
	for _, campaignEmail := range campaignEmails {
		campaignIDsToEmails[campaignEmail.GetCampaignID()] = append(campaignIDsToEmails[campaignEmail.GetCampaignID()], campaignEmail)
	}

	for _, campaign := range campaigns {
		campaign.CampaignEmails = campaignIDsToEmails[campaign.GetID()]
	}

	return campaigns, pNew, nil
}

func (r *campaignRepo) get(ctx context.Context, tenantID uint64, conditions []*Condition, filterDelete bool) (*entity.Campaign, error) {
	campaign := new(Campaign)

	if err := r.baseRepo.Get(ctx, campaign, &Filter{
		Conditions: append(r.getBaseConditions(tenantID), r.mayAddDeleteFilter(conditions, filterDelete)...),
	}); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}

	c := ToCampaign(campaign)

	campaignEmails, err := r.getCampaignEmails(ctx, c.GetID())
	if err != nil {
		return nil, err
	}

	c.CampaignEmails = campaignEmails

	return c, nil
}

func (r *campaignRepo) getCampaignEmails(ctx context.Context, campaignIDs ...uint64) ([]*entity.CampaignEmail, error) {
	res, _, err := r.baseRepo.GetMany(ctx, new(CampaignEmail), &Filter{
		Conditions: []*Condition{
			{
				Field: "campaign_id",
				Value: campaignIDs,
				Op:    OpIn,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	campaignEmails := make([]*entity.CampaignEmail, 0, len(res))
	for _, m := range res {
		campaignEmails = append(campaignEmails, ToCampaignEmail(m.(*CampaignEmail)))
	}

	return campaignEmails, nil
}

func (r *campaignRepo) mayAddDeleteFilter(conditions []*Condition, filterDelete bool) []*Condition {
	if filterDelete {
		return append(conditions, &Condition{
			Field: "status",
			Value: entity.CampaignStatusDeleted,
			Op:    OpNotEq,
		})
	}
	return conditions
}

func (r *campaignRepo) getBaseConditions(tenantID uint64) []*Condition {
	return []*Condition{
		{
			Field:         "tenant_id",
			Value:         tenantID,
			Op:            OpEq,
			NextLogicalOp: LogicalOpAnd,
		},
	}
}

func ToCampaign(campaign *Campaign) *entity.Campaign {
	return &entity.Campaign{
		ID:             campaign.ID,
		Name:           campaign.Name,
		CampaignDesc:   campaign.CampaignDesc,
		SegmentID:      campaign.SegmentID,
		SegmentSize:    campaign.SegmentSize,
		Schedule:       campaign.Schedule,
		Progress:       campaign.Progress,
		TenantID:       campaign.TenantID,
		CreatorID:      campaign.CreatorID,
		Status:         entity.CampaignStatus(campaign.GetStatus()),
		CampaignEmails: nil,
		CreateTime:     campaign.CreateTime,
		UpdateTime:     campaign.UpdateTime,
	}
}

func ToCampaignModel(campaign *entity.Campaign) *Campaign {
	return &Campaign{
		ID:           campaign.ID,
		Name:         campaign.Name,
		CampaignDesc: campaign.CampaignDesc,
		SegmentID:    campaign.SegmentID,
		SegmentSize:  campaign.SegmentSize,
		Schedule:     campaign.Schedule,
		Progress:     campaign.Progress,
		Status:       goutil.Uint32(uint32(campaign.Status)),
		TenantID:     campaign.TenantID,
		CreatorID:    campaign.CreatorID,
		CreateTime:   campaign.CreateTime,
		UpdateTime:   campaign.UpdateTime,
	}
}

func ToCampaignEmail(campaignEmail *CampaignEmail) *entity.CampaignEmail {
	return &entity.CampaignEmail{
		ID:             campaignEmail.ID,
		CampaignID:     campaignEmail.CampaignID,
		EmailID:        campaignEmail.EmailID,
		Subject:        campaignEmail.Subject,
		Ratio:          campaignEmail.Ratio,
		CampaignResult: nil,
	}
}

func ToCampaignEmailModel(campaignID uint64, campaignEmail *entity.CampaignEmail) *CampaignEmail {
	return &CampaignEmail{
		CampaignID: goutil.Uint64(campaignID),
		EmailID:    campaignEmail.EmailID,
		Subject:    campaignEmail.Subject,
		Ratio:      campaignEmail.Ratio,
	}
}
