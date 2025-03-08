package entity

import (
	"cdp/pkg/goutil"
	"time"
)

type CampaignStatus uint32

const (
	CampaignStatusUnknown CampaignStatus = iota
	CampaignStatusPending
	CampaignStatusRunning
	CampaignStatusFailed
	CampaignStatusDeleted
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
	CreatorID      *uint64          `json:"creator_id,omitempty"`
	TenantID       *uint64          `json:"tenant_id,omitempty"`
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

func (e *Campaign) GetTenantID() uint64 {
	if e != nil && e.TenantID != nil {
		return *e.TenantID
	}
	return 0
}

func (e *Campaign) GetSegmentID() uint64 {
	if e != nil && e.SegmentID != nil {
		return *e.SegmentID
	}
	return 0
}

func (e *Campaign) GetStatus() CampaignStatus {
	if e != nil {
		return e.Status
	}
	return CampaignStatusUnknown
}

func (e *Campaign) GetSegmentSize() uint64 {
	if e != nil && e.SegmentSize != nil {
		return *e.SegmentSize
	}
	return 0
}

func (e *Campaign) GetProgress() uint64 {
	if e != nil && e.Progress != nil {
		return *e.Progress
	}
	return 0
}

func (e *Campaign) Update(newCampaign *Campaign) bool {
	var hasChange bool

	if newCampaign.Progress != nil && e.GetProgress() != newCampaign.GetProgress() {
		hasChange = true
		e.Progress = newCampaign.Progress
	}

	if newCampaign.SegmentSize != nil && e.GetSegmentSize() != newCampaign.GetSegmentSize() {
		hasChange = true
		e.SegmentSize = newCampaign.SegmentSize
	}

	if newCampaign.Status != CampaignStatusUnknown && e.Status != newCampaign.Status {
		hasChange = true
		e.Status = newCampaign.Status
	}

	if hasChange {
		e.UpdateTime = goutil.Uint64(uint64(time.Now().Unix()))
	}

	return hasChange
}

type CampaignResult struct {
	TotalUniqueOpenCount *uint64           `json:"total_unique_open_count,omitempty"`
	TotalClickCount      *uint64           `json:"total_click_count,omitempty"`
	AvgOpenTime          *uint64           `json:"avg_open_time,omitempty"`
	ClickCountsByLink    map[string]uint64 `json:"click_counts_by_link"`
}

type CampaignEmail struct {
	ID         *uint64 `json:"id,omitempty"`
	CampaignID *uint64 `json:"campaign_id,omitempty"`
	EmailID    *uint64 `json:"email_id,omitempty"`
	Subject    *string `json:"subject,omitempty"`
	Ratio      *uint64 `json:"ratio,omitempty"`

	Email          *Email          `json:"email,omitempty"`
	CampaignResult *CampaignResult `json:"campaign_result,omitempty"`
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

func (e *CampaignEmail) GetEmailID() uint64 {
	if e != nil && e.EmailID != nil {
		return *e.EmailID
	}
	return 0
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
