package handler

import (
	"cdp/entity"
	"cdp/pkg/validator"
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

var ContextInfoValidator = validator.MustForm(map[string]validator.Validator{
	"user":   UserValidator,
	"tenant": TenantValidator,
})

var UserValidator = validator.MustForm(map[string]validator.Validator{
	"id": &validator.UInt64{
		Optional: false,
	},
})

var TenantValidator = validator.MustForm(map[string]validator.Validator{
	"id": &validator.UInt64{
		Optional: false,
	},
})
