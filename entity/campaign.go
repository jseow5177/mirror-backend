package entity

type CampaignStatus uint32

const (
	CampaignStatusUnknown CampaignStatus = iota
	CampaignStatusPending
	CampaignStatusRunning
	CampaignStatusFailed
)

type CampaignEmail struct {
	ID          *uint64           `json:"id,omitempty"`
	CampaignID  *uint64           `json:"campaign_id,omitempty"`
	EmailID     *uint64           `json:"email_id,omitempty"`
	Subject     *string           `json:"subject,omitempty"`
	Ratio       *uint64           `json:"ratio,omitempty"`
	OpenCount   *uint64           `json:"open_count,omitempty"`
	ClickCounts map[string]uint64 `json:"click_counts,omitempty"`
}

func (e *CampaignEmail) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *CampaignEmail) GetCampaignID() uint64 {
	if e != nil && e.CampaignID != nil {
		return *e.CampaignID
	}
	return 0
}

func (e *CampaignEmail) GetOpenCount() uint64 {
	if e != nil && e.OpenCount != nil {
		return *e.OpenCount
	}
	return 0
}

func (e *CampaignEmail) GetClickCounts() map[string]uint64 {
	if e != nil && e.ClickCounts != nil {
		return e.ClickCounts
	}
	return nil
}

func (e *CampaignEmail) GetRatio() uint64 {
	if e != nil && e.Ratio != nil {
		return *e.Ratio
	}
	return 0
}

func (e *CampaignEmail) GetSubject() string {
	if e != nil && e.Subject != nil {
		return *e.Subject
	}
	return ""
}

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
