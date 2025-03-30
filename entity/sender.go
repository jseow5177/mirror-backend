package entity

import (
	"cdp/pkg/goutil"
	"fmt"
)

type Sender struct {
	ID         *uint64 `json:"id,omitempty"`
	TenantID   *uint64 `json:"tenant_id,omitempty"`
	Name       *string `json:"name,omitempty"`
	LocalPart  *string `json:"local_part,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`

	Email *string `json:"email,omitempty"`
}

func (s *Sender) GetLocalPart() string {
	if s != nil && s.LocalPart != nil {
		return *s.LocalPart
	}
	return ""
}

func (s *Sender) GetName() string {
	if s != nil && s.Name != nil {
		return *s.Name
	}
	return ""
}

func (s *Sender) GetEmail(tenant *Tenant) string {
	return fmt.Sprintf("%s@%s", s.GetLocalPart(), tenant.ExtInfo.GetDomain())
}

func (s *Sender) SetEmail(tenant *Tenant) {
	s.Email = goutil.String(s.GetEmail(tenant))
}
