package entity

import (
	"encoding/json"
	"fmt"
)

type IDType uint32

const (
	IDTypeUnknown IDType = iota
	IDTypeEmail
)

var IDTypes = map[IDType]string{
	IDTypeEmail: "email",
}

type Ud struct {
	ID     *string `json:"id,omitempty"`
	IDType IDType  `json:"id_type,omitempty"`
}

func (e *Ud) GetID() string {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return ""
}

func (e *Ud) GetIDType() IDType {
	if e != nil {
		return e.IDType
	}
	return IDTypeUnknown
}

type TagVal struct {
	TagID  *uint64     `json:"tag_id,omitempty"`
	TagVal interface{} `json:"tag_val,omitempty"`
}

func (e *TagVal) GetTagID() uint64 {
	if e != nil && e.TagID != nil {
		return *e.TagID
	}
	return 0
}

func (e *TagVal) GetTagVal() interface{} {
	if e != nil && e.TagVal != nil {
		return e.TagVal
	}
	return nil
}

type UdTagVal struct {
	Ud      *Ud       `json:"ud,omitempty"`
	TagVals []*TagVal `json:"tag_vals,omitempty"`
}

func (e *UdTagVal) ToDocID() string {
	if e == nil || e.Ud == nil {
		return ""
	}
	return fmt.Sprintf("%s:%d", e.Ud.GetID(), e.Ud.GetIDType())
}

func (e *UdTagVal) ToDoc() (string, error) {
	if e == nil || e.Ud == nil {
		return "", nil
	}

	tagVals := make(map[string]interface{})
	for _, tagVal := range e.TagVals {
		tagVals[fmt.Sprintf("tag_%d", tagVal.GetTagID())] = tagVal.GetTagVal()
	}

	b, err := json.Marshal(tagVals)
	if err != nil {
		return "", err
	}

	return string(b), nil
}
