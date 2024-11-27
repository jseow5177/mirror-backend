package entity

type CampaignStatus uint32

const (
	CampaignStatusUnknown CampaignStatus = iota
	CampaignStatusPending
	CampaignStatusNormal
	CampaignStatusRunning
	CampaignStatusFailed
	CampaignStatusDone
	CampaignStatusDeleted
)

type CampaignEmail struct {
	EmailID *uint64  `json:"email_id,omitempty"`
	Subject *string  `json:"subject,omitempty"`
	Ratio   *float64 `json:"ratio,omitempty"`
}

type CampaignMeta struct {
	Emails []*CampaignEmail `json:"emails,omitempty"`
}

type Campaign struct {
	ID           *uint64        `json:"id,omitempty"`
	Name         *string        `json:"name,omitempty"`
	CampaignDesc *string        `json:"campaign_desc,omitempty"`
	SegmentID    *uint64        `json:"segment_id,omitempty"`
	Status       CampaignStatus `json:"status,omitempty"`
	Meta         *CampaignMeta  `json:"meta,omitempty"`
	CreateTime   *uint64        `json:"create_time,omitempty"`
	UpdateTime   *uint64        `json:"update_time,omitempty"`
}
