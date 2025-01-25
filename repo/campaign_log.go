package repo

import (
	"cdp/entity"
	"cdp/pkg/goutil"
	"context"
	"math"
)

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
	CreateMany(ctx context.Context, campaignLogs []*entity.CampaignLog) error
	CountTotalUniqueOpen(ctx context.Context, campaignEmailID uint64) (uint64, error)
	CountClicksByLink(ctx context.Context, campaignEmailID uint64) (map[string]uint64, error)
	GetAvgOpenTime(ctx context.Context, campaignEmailID uint64) (uint64, error)
}

type campaignLogRepo struct {
	baseRepo BaseRepo
}

func NewCampaignLogRepo(_ context.Context, baseRepo BaseRepo) CampaignLogRepo {
	return &campaignLogRepo{baseRepo: baseRepo}
}

func (r *campaignLogRepo) CreateMany(ctx context.Context, campaignLogs []*entity.CampaignLog) error {
	campaignLogModels := make([]*CampaignLog, 0, len(campaignLogs))
	for _, campaignLog := range campaignLogs {
		campaignLogModels = append(campaignLogModels, ToCampaignLogModel(campaignLog))
	}

	return r.baseRepo.CreateMany(ctx, new(CampaignLog), campaignLogModels)
}

func (r *campaignLogRepo) CountTotalUniqueOpen(ctx context.Context, campaignEmailID uint64) (uint64, error) {
	return r.baseRepo.Count(ctx, new(CampaignLog), &Filter{
		Conditions: []*Condition{
			{
				Field:         "campaign_email_id",
				Value:         campaignEmailID,
				Op:            OpEq,
				NextLogicalOp: LogicalOpAnd,
			},
			{
				Field: "event",
				Value: entity.EventUniqueOpened,
				Op:    OpEq,
			},
		},
	})
}

func (r *campaignLogRepo) GetAvgOpenTime(ctx context.Context, campaignEmailID uint64) (uint64, error) {
	avgOpenTime, err := r.baseRepo.Avg(ctx, new(CampaignLog), "event_time", &Filter{
		Conditions: []*Condition{
			{
				Field:         "campaign_email_id",
				Value:         campaignEmailID,
				Op:            OpEq,
				NextLogicalOp: LogicalOpAnd,
			},
			{
				Field: "event",
				Value: entity.EventUniqueOpened,
				Op:    OpEq,
			},
		},
	})
	if err != nil {
		return 0, err
	}

	return uint64(math.Round(avgOpenTime)), nil
}

type LinkCount struct {
	Link  string
	Count uint64
}

func (r *campaignLogRepo) CountClicksByLink(ctx context.Context, campaignEmailID uint64) (map[string]uint64, error) {
	aggregateFields := map[string]string{
		"link":  "link",
		"count": "COUNT(*)",
	}
	groupByFields := []string{"link"}
	filter := &Filter{
		Conditions: []*Condition{
			{
				Field:         "campaign_email_id",
				Value:         campaignEmailID,
				Op:            OpEq,
				NextLogicalOp: LogicalOpAnd,
			},
			{
				Field: "event",
				Value: entity.EventClick,
				Op:    OpEq,
			},
		},
	}

	res, err := r.baseRepo.GroupBy(ctx, new(CampaignLog), new(LinkCount), groupByFields, aggregateFields, filter)
	if err != nil {
		return nil, err
	}

	linkCounts := make(map[string]uint64)
	for _, r := range res {
		linkCount := r.(*LinkCount)
		linkCounts[linkCount.Link] += linkCount.Count
	}

	return linkCounts, nil
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
