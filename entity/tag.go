package entity

import (
	"cdp/pkg/goutil"
	"encoding/json"
	"errors"
	"strconv"
)

var (
	ErrInvalidTagValueType = errors.New("invalid tag value type")
)

type TagStatus uint32

const (
	TagStatusUnknown TagStatus = iota
	TagStatusNormal
	TagStatusDeleted
)

type TagValueType uint32

const (
	TagValueTypeUnknown TagValueType = iota
	TagValueTypeInt
	TagValueTypeStr
	TagValueTypeFloat
)

var TagValueTypes = map[uint32]string{
	uint32(TagValueTypeInt):   "Int",
	uint32(TagValueTypeStr):   "Str",
	uint32(TagValueTypeFloat): "Float",
}

func CheckTagValueType(value uint32) error {
	_, ok := TagValueTypes[value]
	if !ok {
		return ErrInvalidTagValueType
	}
	return nil
}

type TagExtInfo struct{}

func (e *TagExtInfo) ToString() (string, error) {
	if e == nil {
		return "{}", nil
	}

	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type Tag struct {
	ID         *uint64     `json:"id,omitempty"`
	Name       *string     `json:"name,omitempty"`
	Desc       *string     `json:"desc,omitempty"`
	Enum       []string    `json:"enum,omitempty"`
	ValueType  *uint32     `json:"value_type,omitempty"`
	Status     *uint32     `json:"status,omitempty"`
	ExtInfo    *TagExtInfo `json:"ext_info,omitempty"`
	CreateTime *uint64     `json:"create_time,omitempty"`
	UpdateTime *uint64     `json:"update_time,omitempty"`
}

func (e *Tag) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Tag) GetName() string {
	if e != nil && e.Name != nil {
		return *e.Name
	}
	return ""
}

func (e *Tag) GetDesc() string {
	if e != nil && e.Desc != nil {
		return *e.Desc
	}
	return ""
}

func (e *Tag) GetEnum() []string {
	if e != nil && e.Enum != nil {
		return e.Enum
	}
	return nil
}

func (e *Tag) GetValueType() uint32 {
	if e != nil && e.ValueType != nil {
		return *e.ValueType
	}
	return uint32(TagValueTypeUnknown)
}

func (e *Tag) GetStatus() uint32 {
	if e != nil && e.Status != nil {
		return *e.Status
	}
	return uint32(TagStatusUnknown)
}

func (e *Tag) GetExtInfo() *TagExtInfo {
	if e != nil && e.ExtInfo != nil {
		return e.ExtInfo
	}
	return nil
}

func (e *Tag) GetCreateTime() uint64 {
	if e != nil && e.CreateTime != nil {
		return *e.CreateTime
	}
	return 0
}

func (e *Tag) GetUpdateTime() uint64 {
	if e != nil && e.UpdateTime != nil {
		return *e.UpdateTime
	}
	return 0
}

func (e *Tag) InEnum(tagValue string) bool {
	if len(e.GetEnum()) > 0 && !goutil.ContainsStr(e.GetEnum(), tagValue) {
		return false
	}
	return true
}

func (e *Tag) IsNumeric() bool {
	return goutil.ContainsUint32([]uint32{
		uint32(TagValueTypeInt),
		uint32(TagValueTypeFloat),
	}, e.GetValueType())
}

func (e *Tag) IsFloat() bool {
	return e.GetValueType() == uint32(TagValueTypeFloat)
}

func (e *Tag) IsValidTagValue(value string) bool {
	switch e.GetValueType() {
	case uint32(TagValueTypeStr):
	case uint32(TagValueTypeInt):
		if _, err := strconv.Atoi(value); err != nil {
			return false
		}
	case uint32(TagValueTypeFloat):
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return false
		}
	default:
		return false
	}
	return true
}
