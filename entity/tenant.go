package entity

type TenantStatus uint32

const (
	TenantStatusUnknown TenantStatus = iota
	TenantStatusPending
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

func (e *Tenant) Update(t *Tenant) bool {
	var hasChange bool

	if e.Status != t.Status {
		hasChange = true
		e.Status = t.Status
	}

	return hasChange
}

func (e *Tenant) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Tenant) GetStatus() TenantStatus {
	if e != nil {
		return e.Status
	}
	return TenantStatusUnknown
}

func (e *Tenant) GetName() string {
	if e != nil && e.Name != nil {
		return *e.Name
	}
	return ""
}

func (e *Tenant) IsNormal() bool {
	if e != nil {
		return e.Status == TenantStatusNormal
	}
	return false
}

func (e *Tenant) IsPending() bool {
	if e != nil {
		return e.Status == TenantStatusPending
	}
	return false
}
