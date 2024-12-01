package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
	"errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type CampaignEmail struct {
	ID          *uint64
	CampaignID  *uint64
	EmailID     *uint64
	Subject     *string
	Html        *string
	Ratio       *uint64
	OpenCount   *uint64
	ClickCounts *string
}

func (m *CampaignEmail) GetClickCounts() string {
	if m != nil && m.ClickCounts != nil {
		return *m.ClickCounts
	}
	return ""
}

type CampaignEmailFilter struct {
	ID *uint64
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
	Progress     *uint64
	Status       *uint32
	CreateTime   *uint64
	UpdateTime   *uint64
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
	GetCampaignEmail(_ context.Context, f *CampaignEmailFilter) (*entity.CampaignEmail, error)
	UpdateCampaignEmail(_ context.Context, f *CampaignEmailFilter, campaignEmail *entity.CampaignEmail) error
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

	return ToCampaignEmail(campaignEmail)
}

func (r *campaignRepo) UpdateCampaignEmail(_ context.Context, f *CampaignEmailFilter, campaignEmail *entity.CampaignEmail) error {
	campaignEmailModel, err := ToCampaignEmailModel(campaignEmail.GetCampaignID(), campaignEmail)
	if err != nil {
		return err
	}
	return r.orm.Model(campaignEmailModel).Where(f).Updates(campaignEmailModel).Error
}

func (r *campaignRepo) Create(_ context.Context, campaign *entity.Campaign) (uint64, error) {
	campaignModel := ToCampaignModel(campaign)
	if err := r.orm.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&campaignModel).Error; err != nil {
			return err
		}

		campaign.ID = campaignModel.ID

		for _, campaignEmail := range campaign.CampaignEmails {
			campaignEmailModel, err := ToCampaignEmailModel(campaignModel.GetID(), campaignEmail)
			if err != nil {
				return err
			}

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

func ToCampaignEmail(campaignEmail *CampaignEmail) (*entity.CampaignEmail, error) {
	clickCounts := make(map[string]uint64)
	if err := json.Unmarshal([]byte(campaignEmail.GetClickCounts()), &clickCounts); err != nil {
		return nil, err
	}

	return &entity.CampaignEmail{
		ID:          campaignEmail.ID,
		CampaignID:  campaignEmail.CampaignID,
		EmailID:     campaignEmail.EmailID,
		Subject:     campaignEmail.Subject,
		Html:        campaignEmail.Html,
		Ratio:       campaignEmail.Ratio,
		OpenCount:   campaignEmail.OpenCount,
		ClickCounts: clickCounts,
	}, nil
}

func ToCampaignEmailModel(campaignID uint64, campaignEmail *entity.CampaignEmail) (*CampaignEmail, error) {
	clickCounts := config.EmptyJson
	if campaignEmail.ClickCounts != nil {
		var err error
		clickCounts, err = json.Marshal(campaignEmail.ClickCounts)
		if err != nil {
			return nil, err
		}
	}

	return &CampaignEmail{
		CampaignID:  goutil.Uint64(campaignID),
		EmailID:     campaignEmail.EmailID,
		Subject:     campaignEmail.Subject,
		Html:        campaignEmail.Html,
		Ratio:       campaignEmail.Ratio,
		OpenCount:   campaignEmail.OpenCount,
		ClickCounts: goutil.String(string(clickCounts)),
	}, nil
}

func ToCampaignModel(campaign *entity.Campaign) *Campaign {
	return &Campaign{
		ID:           campaign.ID,
		Name:         campaign.Name,
		CampaignDesc: campaign.CampaignDesc,
		SegmentID:    campaign.SegmentID,
		SegmentSize:  campaign.SegmentSize,
		Progress:     campaign.Progress,
		Status:       goutil.Uint32(uint32(campaign.Status)),
		CreateTime:   campaign.CreateTime,
		UpdateTime:   campaign.UpdateTime,
	}
}
