package entity

type TenantStatus uint32

const (
	TenantStatusUnknown TenantStatus = iota
	TenantStatusNormal
	TenantStatusDeleted
)

type Tenant struct {
	ID         *uint64      `json:"id,omitempty"`
	Name       *string      `json:"name,omitempty"`
	Status     TenantStatus `json:"status,omitempty"`
	CreateTime *uint64      `json:"create_time,omitempty"`
	UpdateTime *uint64      `json:"update_time,omitempty"`
}
