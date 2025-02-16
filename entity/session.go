package entity

import (
	"cdp/config"
	"cdp/pkg/goutil"
	"time"
)

const (
	sessionByteLength = 32
)

type Session struct {
	ID         *uint64 `json:"id,omitempty"`
	UserID     *uint64 `json:"user_id,omitempty"`
	Token      *string `json:"token,omitempty"`
	TokenHash  *string `json:"-"`
	ExpireTime *uint64 `json:"expire_time,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
}

func (e *Session) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Session) GetTokenHash() string {
	if e != nil && e.TokenHash != nil {
		return *e.TokenHash
	}
	return ""
}

func (e *Session) GetUserID() uint64 {
	if e != nil && e.UserID != nil {
		return *e.UserID
	}
	return 0
}

func (e *Session) GetToken() string {
	if e != nil && e.Token != nil {
		return *e.Token
	}
	return ""
}

func NewSession(userID uint64) (*Session, error) {
	now := time.Now()
	expire := now.Add(config.ThreeMonths) // TODO: 3 months

	token, err := goutil.GenerateSecureRandString(sessionByteLength)
	if err != nil {
		return nil, err
	}

	return &Session{
		UserID:     goutil.Uint64(userID),
		Token:      goutil.String(goutil.Base64Encode(token)),
		TokenHash:  goutil.String(goutil.Sha256(token)),
		CreateTime: goutil.Uint64(uint64(now.Unix())),
		ExpireTime: goutil.Uint64(uint64(expire.Unix())),
	}, nil
}
