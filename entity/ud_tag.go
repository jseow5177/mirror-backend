package entity

import (
	"cdp/pkg/goutil"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

const docIDSeparator = ":"

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

func (e *Ud) ToDocID() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("%s%s%d", e.GetID(), docIDSeparator, e.GetIDType())
}

func ToUd(docID string) (*Ud, error) {
	parts := strings.Split(docID, docIDSeparator)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid doc id format: %s", docID)
	}

	i, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid id type, docID: %s, err: %v", docID, err)
	}

	idType := IDType(i)
	if idType == IDTypeUnknown {
		return nil, fmt.Errorf("id type should not be Unknown, docID: %s, err: %v", docID, err)
	}

	return &Ud{
		ID:     goutil.String(parts[0]),
		IDType: idType,
	}, nil
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

func (e *UdTagVal) GetUd() *Ud {
	if e == nil {
		return nil
	}
	return e.Ud
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
