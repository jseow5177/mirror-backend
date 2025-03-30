package entity

import (
	"bytes"
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
	FolderID      string                            `json:"folder_id,omitempty"`
	Domain        string                            `json:"domain,omitempty"`
	DnsRecords    map[string]map[string]interface{} `json:"dns_records,omitempty"`
	IsDomainValid *bool                             `json:"is_domain_valid,omitempty"`
}

func (e *TenantExtInfo) IsDnsRecordsEqual(other *TenantExtInfo) bool {
	aBytes, _ := json.Marshal(e.DnsRecords)
	bBytes, _ := json.Marshal(other.DnsRecords)
	return bytes.Equal(aBytes, bBytes)
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

func (e *TenantExtInfo) GetDomain() string {
	return e.Domain
}

func (e *TenantExtInfo) GetIsDomainValid() bool {
	if e != nil && e.IsDomainValid != nil {
		return *e.IsDomainValid
	}
	return false
}

type Tenant struct {
	ID         *uint64        `json:"id,omitempty"`
	Name       *string        `json:"name,omitempty"`
	Status     TenantStatus   `json:"status,omitempty"`
	ExtInfo    *TenantExtInfo `json:"ext_info,omitempty"`
	CreateTime *uint64        `json:"create_time,omitempty"`
	UpdateTime *uint64        `json:"update_time,omitempty"`
}

func (e *Tenant) Update(newTenant *Tenant) bool {
	var hasChange bool

	if newTenant.Status != TenantStatusUnknown && e.Status != newTenant.Status {
		hasChange = true
		e.Status = newTenant.Status
	}

	if newTenant.ExtInfo != nil {
		if e.ExtInfo == nil {
			e.ExtInfo = new(TenantExtInfo)
		}

		if newTenant.ExtInfo.FolderID != "" && e.ExtInfo.FolderID != newTenant.ExtInfo.FolderID {
			hasChange = true
			e.ExtInfo.FolderID = newTenant.ExtInfo.FolderID
		}

		if newTenant.ExtInfo.Domain != "" && e.ExtInfo.Domain != newTenant.ExtInfo.Domain {
			hasChange = true
			e.ExtInfo.Domain = newTenant.ExtInfo.Domain
		}

		if newTenant.ExtInfo.DnsRecords != nil {
			if e.ExtInfo.DnsRecords == nil || !newTenant.ExtInfo.IsDnsRecordsEqual(e.ExtInfo) {
				hasChange = true
				e.ExtInfo.DnsRecords = newTenant.ExtInfo.DnsRecords
			}
		}

		if newTenant.ExtInfo.IsDomainValid != nil && e.ExtInfo.GetIsDomainValid() != newTenant.ExtInfo.GetIsDomainValid() {
			hasChange = true
			e.ExtInfo.IsDomainValid = newTenant.ExtInfo.IsDomainValid
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
