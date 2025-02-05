package entity

import (
	"cdp/pkg/goutil"
	"fmt"
	"strings"
	"time"
)

type UserStatus uint32

const (
	UserStatusUnknown UserStatus = iota
	UserStatusPending
	UserStatusNormal
	UserStatusDeleted
)

type User struct {
	ID          *uint64    `json:"id,omitempty"`
	TenantID    *uint64    `json:"tenant_id,omitempty"`
	Email       *string    `json:"email,omitempty"`
	Username    *string    `json:"username,omitempty"`
	Password    *string    `json:"-"`
	DisplayName *string    `json:"display_name,omitempty"`
	Status      UserStatus `json:"status,omitempty"`
	CreateTime  *uint64    `json:"create_time,omitempty"`
	UpdateTime  *uint64    `json:"update_time,omitempty"`
}

func NewUser(tenantID uint64, email string, password string, displayName string) (*User, error) {
	now := uint64(time.Now().Unix())

	username, err := extractUsernameFromEmail(email)
	if err != nil {
		return nil, err
	}

	var (
		passwordHash string
		status       = UserStatusPending
	)
	if password != "" {
		passwordHash, err = hashPassword(password)
		if err != nil {
			return nil, err
		}
		status = UserStatusNormal
	}

	return &User{
		TenantID:    goutil.Uint64(tenantID),
		Email:       goutil.String(email),
		Username:    goutil.String(username),
		Password:    goutil.String(passwordHash),
		DisplayName: goutil.String(displayName),
		Status:      status,
		CreateTime:  goutil.Uint64(now),
		UpdateTime:  goutil.Uint64(now),
	}, nil
}

func (e *User) Update(t *User) bool {
	var hasChange bool

	if t.Status != UserStatusUnknown && e.Status != t.Status {
		hasChange = true
		e.Status = t.Status
	}

	if t.Password != nil && e.GetPassword() != t.GetPassword() {
		hasChange = true
		e.Password = t.Password
	}

	if hasChange {
		e.UpdateTime = goutil.Uint64(uint64(time.Now().Unix()))
	}

	return hasChange
}

func (e *User) ComparePassword(input string) bool {
	return goutil.CompareBCrypt(e.GetPassword(), input) == nil
}

func (e *User) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *User) GetTenantID() uint64 {
	if e != nil && e.TenantID != nil {
		return *e.TenantID
	}
	return 0
}

func (e *User) GetDisplayName() string {
	if e != nil && e.DisplayName != nil {
		return *e.DisplayName
	}
	return ""
}

func (e *User) GetStatus() UserStatus {
	if e != nil {
		return e.Status
	}
	return UserStatusUnknown
}

func (e *User) GetEmail() string {
	if e != nil && e.Email != nil {
		return *e.Email
	}
	return ""
}

func (e *User) GetPassword() string {
	if e != nil && e.Password != nil {
		return *e.Password
	}
	return ""
}

func (e *User) GetUsername() string {
	if e != nil && e.Username != nil {
		return *e.Username
	}
	return ""
}

func (e *User) IsPending() bool {
	return e.GetStatus() == UserStatusPending
}

func extractUsernameFromEmail(email string) (string, error) {
	parts := strings.Split(email, "@")
	if len(parts) != 2 || parts[0] == "" {
		return "", fmt.Errorf("invalid email: %v", email)
	}
	return parts[0], nil
}

func hashPassword(password string) (string, error) {
	return goutil.BCrypt(password)
}
