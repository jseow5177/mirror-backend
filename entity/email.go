package entity

import (
	"cdp/pkg/goutil"
	"time"
)

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
	CreatorID  *uint64     `json:"creator_id,omitempty"`
	TenantID   *uint64     `json:"tenant_id,omitempty"`
	CreateTime *uint64     `json:"create_time,omitempty"`
	UpdateTime *uint64     `json:"update_time,omitempty"`
}

func (e *Email) GetName() string {
	if e != nil && e.Name != nil {
		return *e.Name
	}
	return ""
}

func (e *Email) GetEmailDesc() string {
	if e != nil && e.EmailDesc != nil {
		return *e.EmailDesc
	}
	return ""
}

func (e *Email) GetJson() string {
	if e != nil && e.Json != nil {
		return *e.Json
	}
	return ""
}

func (e *Email) GetStatus() EmailStatus {
	if e != nil {
		return e.Status
	}
	return EmailStatusUnknown
}

func (e *Email) GetHtml() string {
	if e != nil && e.Html != nil {
		return *e.Html
	}
	return ""
}

func (e *Email) Update(newEmail *Email) bool {
	var hasChange bool

	if newEmail.Name != nil && newEmail.GetName() != e.GetName() {
		hasChange = true
		e.Name = newEmail.Name
	}

	if newEmail.EmailDesc != nil && newEmail.GetEmailDesc() != e.GetEmailDesc() {
		hasChange = true
		e.EmailDesc = newEmail.EmailDesc
	}

	if newEmail.Json != nil && newEmail.GetJson() != e.GetJson() {
		hasChange = true
		e.Json = newEmail.Json
	}

	if newEmail.Html != nil && newEmail.GetHtml() != e.GetHtml() {
		hasChange = true
		e.Html = newEmail.Html
	}

	if hasChange {
		e.UpdateTime = goutil.Uint64(uint64(time.Now().Unix()))
	}

	return hasChange
}
