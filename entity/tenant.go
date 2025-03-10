package entity

import (
	"cdp/pkg/goutil"
	"encoding/json"
	"time"
)

type TenantStatus uint32

const (
	TenantStatusUnknown TenantStatus = iota
	TenantStatusNormal
	TenantStatusDeleted
)

type TenantExtInfo struct {
	FolderID string `json:"folder_id,omitempty"`
}

func (e *TenantExtInfo) ToString() (string, error) {
	if e == nil {
		return "{}", nil
	}

	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (e *TenantExtInfo) GetFolderID() string {
	return e.FolderID
}

type Tenant struct {
	ID         *uint64        `json:"id,omitempty"`
	Name       *string        `json:"name,omitempty"`
	Status     TenantStatus   `json:"status,omitempty"`
	ExtInfo    *TenantExtInfo `json:"ext_info,omitempty"`
	CreateTime *uint64        `json:"create_time,omitempty"`
	UpdateTime *uint64        `json:"update_time,omitempty"`
}

func (e *Tenant) Update(t *Tenant) bool {
	var hasChange bool

	if t.Status != TenantStatusUnknown && e.Status != t.Status {
		hasChange = true
		e.Status = t.Status
	}

	if t.ExtInfo != nil {
		oldExtInfo := e.ExtInfo
		if oldExtInfo == nil {
			oldExtInfo = new(TenantExtInfo)
		}

		if t.ExtInfo.FolderID != "" && oldExtInfo.FolderID != t.ExtInfo.FolderID {
			hasChange = true
			oldExtInfo.FolderID = t.ExtInfo.FolderID
		}
	}

	if hasChange {
		e.UpdateTime = goutil.Uint64(uint64(time.Now().Unix()))
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

func (e *Tenant) GetExtInfo() *TenantExtInfo {
	if e != nil && e.ExtInfo != nil {
		return e.ExtInfo
	}
	return nil
}

func (e *Tenant) IsNormal() bool {
	if e != nil {
		return e.Status == TenantStatusNormal
	}
	return false
}
