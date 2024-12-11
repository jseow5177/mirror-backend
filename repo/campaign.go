package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type CampaignEmail struct {
	ID         *uint64
	CampaignID *uint64
	EmailID    *uint64
	Subject    *string
	Ratio      *uint64
}

type CampaignEmailFilter struct {
	ID         *uint64
	CampaignID *uint64
}

func (m *CampaignEmail) TableName() string {
	return "campaign_email_tab"
}

var (
	ErrCampaignEmailNotFound = errors.New("campaign email not found")
)

type Campaign struct {
	ID           *uint64
	Name         *string
	CampaignDesc *string
	SegmentID    *uint64
	SegmentSize  *uint64
	Schedule     *uint64
	Progress     *uint64
	Status       *uint32
	CreateTime   *uint64
	UpdateTime   *uint64
}

func (m *Campaign) GetStatus() uint32 {
	if m != nil && m.Status != nil {
		return *m.Status
	}
	return 0
}

type CampaignFilter struct {
	Conditions []*Condition
	Pagination *Pagination
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
	Update(ctx context.Context, f *CampaignFilter, campaign *entity.Campaign) error
	GetMany(ctx context.Context, f *CampaignFilter) ([]*entity.Campaign, *entity.Pagination, error)
	GetCampaignEmail(ctx context.Context, f *CampaignEmailFilter) (*entity.CampaignEmail, error)
	UpdateCampaignEmail(ctx context.Context, f *CampaignEmailFilter, campaignEmail *entity.CampaignEmail) error
	Close(ctx context.Context) error
}

type campaignRepo struct {
	orm *gorm.DB
}

func NewCampaignRepo(_ context.Context, mysqlCfg config.MySQL) (CampaignRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &campaignRepo{orm: orm}, nil
}

func (r *campaignRepo) GetCampaignEmail(_ context.Context, f *CampaignEmailFilter) (*entity.CampaignEmail, error) {
	campaignEmail := new(CampaignEmail)
	if err := r.orm.Where(f).First(campaignEmail).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignEmailNotFound
		}
	}

	return ToCampaignEmail(campaignEmail), nil
}

func (r *campaignRepo) UpdateCampaignEmail(_ context.Context, f *CampaignEmailFilter, campaignEmail *entity.CampaignEmail) error {
	campaignEmailModel := ToCampaignEmailModel(campaignEmail)
	return r.orm.Model(campaignEmailModel).Where(f).Updates(ToCampaignEmailModel(campaignEmail)).Error
}

func (r *campaignRepo) GetManyCampaignEmails(_ context.Context, f *CampaignEmailFilter) ([]*entity.CampaignEmail, error) {
	mCampaignEmails := make([]*CampaignEmail, 0)
	if err := r.orm.Where(f).Find(&mCampaignEmails).Error; err != nil {
		return nil, err
	}

	campaignEmails := make([]*entity.CampaignEmail, len(mCampaignEmails))
	for i, mCampaignEmail := range mCampaignEmails {
		campaignEmails[i] = ToCampaignEmail(mCampaignEmail)
	}

	return campaignEmails, nil
}

func (r *campaignRepo) Update(_ context.Context, f *CampaignFilter, campaign *entity.Campaign) error {
	cond, args := ToSqlWithArgs(f.Conditions)
	campaignModel := ToCampaignModel(campaign)
	return r.orm.Model(campaignModel).Where(cond, args...).Updates(campaignModel).Error
}

func (r *campaignRepo) Create(_ context.Context, campaign *entity.Campaign) (uint64, error) {
	campaignModel := ToCampaignModel(campaign)
	if err := r.orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&campaignModel).Error; err != nil {
			return err
		}

		campaign.ID = campaignModel.ID

		for _, campaignEmail := range campaign.CampaignEmails {
			campaignEmail.CampaignID = campaignModel.ID

			campaignEmailModel := ToCampaignEmailModel(campaignEmail)

			if err := tx.Create(&campaignEmailModel).Error; err != nil {
				return err
			}

			campaignEmail.ID = campaignEmailModel.ID
		}

		return nil
	}); err != nil {
		return 0, err
	}

	return campaignModel.GetID(), nil
}

func (r *campaignRepo) GetMany(ctx context.Context, f *CampaignFilter) ([]*entity.Campaign, *entity.Pagination, error) {
	cond, args := ToSqlWithArgs(f.Conditions)

	var count int64
	if err := r.orm.Model(new(Campaign)).Where(cond, args...).Count(&count).Error; err != nil {
		return nil, nil, err
	}

	var (
		limit = f.Pagination.GetLimit()
		page  = f.Pagination.GetPage()
	)
	if page == 0 {
		page = 1
	}

	var (
		offset     = (page - 1) * limit
		mCampaigns = make([]*Campaign, 0)
	)
	query := r.orm.Where(cond, args...).Offset(int(offset))
	if limit > 0 {
		query = query.Limit(int(limit + 1))
	}

	if err := query.Find(&mCampaigns).Error; err != nil {
		return nil, nil, err
	}

	var hasNext bool
	if limit > 0 && len(mCampaigns) > int(limit) {
		hasNext = true
		mCampaigns = mCampaigns[:limit]
	}

	campaigns := make([]*entity.Campaign, len(mCampaigns))
	for i, mCampaign := range mCampaigns {
		campaign := ToCampaign(mCampaign)

		campaignEmails, err := r.GetManyCampaignEmails(ctx, &CampaignEmailFilter{
			CampaignID: campaign.ID,
		})
		if err != nil {
			return nil, nil, err
		}
		campaign.CampaignEmails = campaignEmails
		campaigns[i] = campaign
	}

	return campaigns, &entity.Pagination{
		Page:    goutil.Uint32(page),
		Limit:   f.Pagination.Limit, // may be nil
		HasNext: goutil.Bool(hasNext),
		Total:   goutil.Int64(count),
	}, nil
}

func (r *campaignRepo) Close(_ context.Context) error {
	if r.orm != nil {
		sqlDB, err := r.orm.DB()
		if err != nil {
			return err
		}

		err = sqlDB.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func ToCampaignEmail(campaignEmail *CampaignEmail) *entity.CampaignEmail {

	return &entity.CampaignEmail{
		ID:         campaignEmail.ID,
		CampaignID: campaignEmail.CampaignID,
		EmailID:    campaignEmail.EmailID,
		Subject:    campaignEmail.Subject,
		Ratio:      campaignEmail.Ratio,
	}
}

func ToCampaignEmailModel(campaignEmail *entity.CampaignEmail) *CampaignEmail {
	return &CampaignEmail{
		CampaignID: campaignEmail.CampaignID,
		EmailID:    campaignEmail.EmailID,
		Subject:    campaignEmail.Subject,
		Ratio:      campaignEmail.Ratio,
	}
}

func ToCampaign(campaign *Campaign) *entity.Campaign {
	return &entity.Campaign{
		ID:           campaign.ID,
		Name:         campaign.Name,
		CampaignDesc: campaign.CampaignDesc,
		SegmentID:    campaign.SegmentID,
		SegmentSize:  campaign.SegmentSize,
		Schedule:     campaign.Schedule,
		Progress:     campaign.Progress,
		Status:       entity.CampaignStatus(campaign.GetStatus()),
		CreateTime:   campaign.CreateTime,
		UpdateTime:   campaign.UpdateTime,
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
		CreateTime:   campaign.CreateTime,
		UpdateTime:   campaign.UpdateTime,
	}
}
