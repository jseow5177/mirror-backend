package entity

type EmailStatus uint32

const (
	EmailStatusUnknown EmailStatus = iota
	EmailStatusNormal
	EmailStatusDeleted
)

type Email struct {
	ID         *uint64     `json:"id,omitempty"`
	Name       *string     `json:"name,omitempty"`
	EmailDesc  *string     `json:"email_desc,omitempty"`
	Json       *string     `json:"json,omitempty"`
	Html       *string     `json:"html,omitempty"`
	Status     EmailStatus `json:"status,omitempty"`
	CreateTime *uint64     `json:"create_time,omitempty"`
	UpdateTime *uint64     `json:"update_time,omitempty"`
}

func (e *Email) GetStatus() EmailStatus {
	if e != nil {
		return e.Status
	}
	return EmailStatusUnknown
}
