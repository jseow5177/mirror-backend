package entity

type CampaignResult struct {
	TotalUniqueOpenCount *uint64           `json:"total_unique_open_count,omitempty"`
	TotalClickCount      *uint64           `json:"total_click_count,omitempty"`
	ClickCountsByLink    map[string]uint64 `json:"click_counts_by_link,omitempty"`
	AvgOpenTime          *uint64           `json:"avg_open_time,omitempty"`
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
