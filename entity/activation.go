package entity

import (
	"cdp/pkg/goutil"
	"time"
)

const (
	tokenByteLength = 32
)

type TokenType uint32

const (
	TokenTypeUnknown TokenType = iota
	TokenTypeUser
	TokenTypeTenant
)

type Activation struct {
	ID         *uint64   `json:"id,omitempty"`
	Token      *string   `json:"token,omitempty"`
	TokenHash  *string   `json:"-"`
	TargetID   *uint64   `json:"target_id,omitempty"`
	TokenType  TokenType `json:"token_type,omitempty"`
	ExpireTime *uint64   `json:"expire_time,omitempty"`
	CreateTime *uint64   `json:"create_time,omitempty"`
}

func (e *Activation) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Activation) GetTokenType() TokenType {
	if e != nil {
		return e.TokenType
	}
	return TokenTypeUnknown
}

func (e *Activation) GetTokenHash() string {
	if e != nil && e.TokenHash != nil {
		return *e.TokenHash
	}
	return ""
}

func (e *Activation) GetToken() string {
	if e != nil && e.Token != nil {
		return *e.Token
	}
	return ""
}

func (e *Activation) GetTargetID() uint64 {
	if e != nil && e.TargetID != nil {
		return *e.TargetID
	}
	return 0
}

func NewActivation(targetID uint64, tokenType TokenType) (*Activation, error) {
	now := uint64(time.Now().Unix())

	token, err := goutil.GenerateRandomString(tokenByteLength)
	if err != nil {
		return nil, err
	}

	return &Activation{
		TargetID:   goutil.Uint64(targetID),
		Token:      goutil.String(goutil.Base64Encode(token)),
		TokenHash:  goutil.String(goutil.Sha256(token)),
		TokenType:  tokenType,
		CreateTime: goutil.Uint64(now),
		ExpireTime: goutil.Uint64(now), // TODO: Add expiry logic
	}, nil
}
