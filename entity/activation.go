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
	ID         *uint64
	Token      *string
	TokenHash  *string
	TargetID   *uint64
	TokenType  TokenType
	ExpireTime *uint64
	CreateTime *uint64
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

func (e *Activation) ToEncodedToken() string {
	return goutil.Base64Encode(e.GetToken())
}

func NewActivation(targetID uint64, tokenType TokenType) (*Activation, error) {
	now := uint64(time.Now().Unix())

	token, err := goutil.GenerateRandomString(tokenByteLength)
	if err != nil {
		return nil, err
	}

	return &Activation{
		TargetID:   goutil.Uint64(targetID),
		Token:      goutil.String(token),
		TokenHash:  goutil.String(goutil.Sha256(token)),
		TokenType:  tokenType,
		CreateTime: goutil.Uint64(now),
		ExpireTime: goutil.Uint64(now), // TODO: Add expiry logic
	}, nil
}
