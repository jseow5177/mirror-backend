package entity

type CampaignStatus uint32

const (
	CampaignStatusUnknown CampaignStatus = iota
	CampaignStatusPending
	CampaignStatusRunning
	CampaignStatusFailed
)

type Campaign struct {
	ID             *uint64          `json:"id,omitempty"`
	Name           *string          `json:"name,omitempty"`
	CampaignDesc   *string          `json:"campaign_desc,omitempty"`
	SegmentID      *uint64          `json:"segment_id,omitempty"`
	SegmentSize    *uint64          `json:"segment_size,omitempty"`
	Progress       *uint64          `json:"progress,omitempty"`
	Status         CampaignStatus   `json:"status,omitempty"`
	CampaignEmails []*CampaignEmail `json:"campaign_emails,omitempty"`
	Schedule       *uint64          `json:"schedule,omitempty"`
	CreateTime     *uint64          `json:"create_time,omitempty"`
	UpdateTime     *uint64          `json:"update_time,omitempty"`

	Segment *Segment `json:"segment,omitempty"`
}

func (e *Campaign) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Campaign) GetStatus() CampaignStatus {
	if e != nil {
		return e.Status
	}
	return CampaignStatusUnknown
}
