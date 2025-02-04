package handler

import (
	"cdp/entity"
)

type ContextInfo struct {
	User   *entity.User
	Tenant *entity.Tenant
}

func (c *ContextInfo) SetUser(u *entity.User) {
	c.User = u
}

func (c *ContextInfo) SetTenant(t *entity.Tenant) {
	c.Tenant = t
}

func (c *ContextInfo) GetUserID() uint64 {
	return c.User.GetID()
}

func (c *ContextInfo) GetTenantID() uint64 {
	return c.Tenant.GetID()
}

func (c *ContextInfo) GetTenantFolder() string {
	return c.Tenant.GetExtInfo().GetFolderID()
}
