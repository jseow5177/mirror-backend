package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type LogExtra struct {
	Link *string
}

type CampaignLog struct {
	ID              *uint64
	CampaignEmailID *uint64
	Event           *uint32
	Link            *string
	Email           *string
	EventTime       *uint64
	CreateTime      *uint64
}

func (m *CampaignLog) TableName() string {
	return "campaign_log_tab"
}

type CampaignLogRepo interface {
	BatchCreate(ctx context.Context, campaignLogs []*entity.CampaignLog) error
	CountTotalUniqueOpen(ctx context.Context, campaignEmailID uint64) (uint64, error)
	CountClicksByLink(ctx context.Context, campaignEmailID uint64) (map[string]uint64, error)
	Close(ctx context.Context) error
}

type campaignLogRepo struct {
	orm *gorm.DB
}

func NewCampaignLogRepo(_ context.Context, mysqlCfg config.MySQL) (CampaignLogRepo, error) {
	orm, err := gorm.Open(mysql.Open(mysqlCfg.ToDSN()), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return &campaignLogRepo{orm: orm}, nil
}

func (r *campaignLogRepo) BatchCreate(_ context.Context, campaignLogs []*entity.CampaignLog) error {
	campaignLogModels := make([]*CampaignLog, 0, len(campaignLogs))
	for _, campaignLog := range campaignLogs {
		campaignLogModels = append(campaignLogModels, ToCampaignLogModel(campaignLog))
	}

	return r.orm.Create(campaignLogModels).Error
}

func (r *campaignLogRepo) CountTotalUniqueOpen(_ context.Context, campaignEmailID uint64) (uint64, error) {
	var count int64
	if err := r.orm.
		Model(new(CampaignLog)).
		Where("campaign_email_id = ? AND event = ?", campaignEmailID, entity.EventUniqueOpened).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return uint64(count), nil
}

func (r *campaignLogRepo) CountClicksByLink(_ context.Context, campaignEmailID uint64) (map[string]uint64, error) {
	rows, err := r.orm.Model(new(CampaignLog)).
		Where("campaign_email_id = ? AND event = ?", campaignEmailID, entity.EventClick).
		Select("link, COUNT(*) as count").
		Group("link").
		Rows()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	linkCounts := make(map[string]uint64)

	var (
		link  string
		count int64
	)
	for rows.Next() {
		err := rows.Scan(&link, &count)
		if err != nil {
			return nil, err
		}
		linkCounts[link] = uint64(count)
	}

	return linkCounts, nil
}

func (r *campaignLogRepo) Close(_ context.Context) error {
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

func ToCampaignLogModel(campaignLog *entity.CampaignLog) *CampaignLog {
	return &CampaignLog{
		CampaignEmailID: campaignLog.CampaignEmailID,
		Event:           goutil.Uint32(uint32(campaignLog.GetEvent())),
		Link:            campaignLog.Link,
		Email:           campaignLog.Email,
		EventTime:       campaignLog.EventTime,
		CreateTime:      campaignLog.CreateTime,
	}
}
