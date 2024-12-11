package repo

import (
	"cdp/config"
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"encoding/json"
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
	LogExtra        *string
	CreateTime      *uint64
}

func (m *CampaignLog) TableName() string {
	return "campaign_log_tab"
}

type CampaignLogRepo interface {
	BatchCreate(ctx context.Context, campaignLogs []*entity.CampaignLog) error
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
		campaignLogModel, err := ToCampaignLogModel(campaignLog)
		if err != nil {
			return err
		}
		campaignLogModels = append(campaignLogModels, campaignLogModel)
	}

	return r.orm.Create(campaignLogModels).Error
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

func ToCampaignLogModel(campaignLog *entity.CampaignLog) (*CampaignLog, error) {
	logExtra, err := json.Marshal(campaignLog.LogExtra)
	if err != nil {
		return nil, err
	}

	return &CampaignLog{
		CampaignEmailID: campaignLog.CampaignEmailID,
		Event:           goutil.Uint32(uint32(campaignLog.GetEvent())),
		LogExtra:        goutil.String(string(logExtra)),
		CreateTime:      campaignLog.CreateTime,
	}, nil
}
