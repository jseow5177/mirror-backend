package entity

import (
	"cdp/pkg/goutil"
	"time"
)

// ActionCode defines the action that a role can carry out.
// This is Mirror's system-wide actions that need IAM control.
type ActionCode string

var (
	ActionUnknown  ActionCode = ""
	ActionEditUser ActionCode = "edit_user"
	ActionEditRole ActionCode = "edit_role"
)

var Actions = map[string][]*Action{
	"User Management": {
		{
			Name:       "Edit User",
			Code:       ActionEditUser,
			ActionDesc: "Can add or delete a user",
		},
		{
			Name:       "Edit Role",
			Code:       ActionEditRole,
			ActionDesc: "Can add, edit, or delete a role and its actions",
		},
	},
}

type Action struct {
	Name       string     `json:"name,omitempty"`
	Code       ActionCode `json:"code,omitempty"`
	ActionDesc string     `json:"action_desc,omitempty"`
}

type RoleStatus uint32

const (
	RoleStatusUnknown RoleStatus = iota
	RoleStatusNormal
	RoleStatusDeleted
)

// Role defines the types of roles, like Admin or Member.
type Role struct {
	ID         *uint64      `json:"id,omitempty"`
	Name       *string      `json:"name,omitempty"`
	RoleDesc   *string      `json:"role_desc,omitempty"`
	Status     RoleStatus   `json:"status,omitempty"`
	Actions    []ActionCode `json:"actions"`
	TenantID   *uint64      `json:"tenant_id,omitempty"`
	CreateTime *uint64      `json:"create_time,omitempty"`
	UpdateTime *uint64      `json:"update_time,omitempty"`
}

func (e *Role) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Role) GetTenantID() uint64 {
	if e != nil && e.TenantID != nil {
		return *e.TenantID
	}
	return 0
}

func (e *Role) GetName() string {
	if e != nil && e.Name != nil {
		return *e.Name
	}
	return ""
}

func (e *Role) GetRoleDesc() string {
	if e != nil && e.RoleDesc != nil {
		return *e.RoleDesc
	}
	return ""
}

func (e *Role) GetStatus() RoleStatus {
	if e != nil {
		return e.Status
	}
	return RoleStatusUnknown
}

func (e *Role) Update(newRole *Role) bool {
	var hasChange bool

	if newRole.Name != nil && e.GetName() != newRole.GetName() {
		hasChange = true
		e.Name = newRole.Name
	}

	if newRole.RoleDesc != nil && e.GetRoleDesc() != newRole.GetRoleDesc() {
		hasChange = true
		e.RoleDesc = newRole.RoleDesc
	}

	var (
		a = make([]string, 0)
		b = make([]string, 0)
	)
	for _, action := range e.Actions {
		a = append(a, string(action))
	}
	for _, action := range newRole.Actions {
		b = append(b, string(action))
	}

	if newRole.Actions != nil && !goutil.IsStrArrEqual(a, b) {
		hasChange = true
		e.Actions = newRole.Actions
	}

	if hasChange {
		e.UpdateTime = goutil.Uint64(uint64(time.Now().Unix()))
	}

	return hasChange
}

// UserRole defines the relationship between users and their roles.
type UserRole struct {
	ID         *uint64 `json:"id,omitempty"`
	TenantID   *uint64 `json:"tenant_id,omitempty"`
	UserID     *uint64 `json:"user_id,omitempty"`
	RoleID     *uint64 `json:"role_id,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}

func (e *UserRole) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *UserRole) GetUserID() uint64 {
	if e != nil && e.UserID != nil {
		return *e.UserID
	}
	return 0
}

func (e *UserRole) GetRoleID() uint64 {
	if e != nil && e.RoleID != nil {
		return *e.RoleID
	}
	return 0
}

func (e *UserRole) GetTenantID() uint64 {
	if e != nil && e.TenantID != nil {
		return *e.TenantID
	}
	return 0
}
