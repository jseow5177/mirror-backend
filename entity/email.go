package entity

type EmailStatus uint32

const (
	EmailStatusUnknown TagStatus = iota
	EmailStatusNormal
	EmailStatusDeleted
)

type Email struct {
	ID         *uint64 `json:"id,omitempty"`
	Name       *string `json:"name,omitempty"`
	EmailDesc  *string `json:"email_desc,omitempty"`
	Blob       *string `json:"blob,omitempty"`
	Status     *uint32 `json:"status,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}
