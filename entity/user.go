package entity

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

func (e *User) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
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
