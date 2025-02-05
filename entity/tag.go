package entity

import (
	"cdp/pkg/goutil"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
)

type floatFrac float64

func (f floatFrac) MarshalJSON() ([]byte, error) {
	n := float64(f)
	if math.IsInf(n, 0) || math.IsNaN(n) {
		return nil, errors.New("unsupported number")
	}

	prec := -1
	if math.Trunc(n) == n {
		prec = 1 // Force ".0" for integers.
	}
	return strconv.AppendFloat(nil, n, 'f', prec, 64), nil
}

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
	ID         *uint64      `json:"id,omitempty"`
	Name       *string      `json:"name,omitempty"`
	TagDesc    *string      `json:"tag_desc,omitempty"`
	Enum       []string     `json:"enum,omitempty"`
	ValueType  TagValueType `json:"value_type,omitempty"`
	Status     TagStatus    `json:"status,omitempty"`
	ExtInfo    *TagExtInfo  `json:"ext_info,omitempty"`
	CreatorID  *uint64      `json:"creator_id,omitempty"`
	TenantID   *uint64      `json:"tenant_id,omitempty"`
	CreateTime *uint64      `json:"create_time,omitempty"`
	UpdateTime *uint64      `json:"update_time,omitempty"`
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

func (e *Tag) GetTagDesc() string {
	if e != nil && e.TagDesc != nil {
		return *e.TagDesc
	}
	return ""
}

func (e *Tag) GetEnum() []string {
	if e != nil && e.Enum != nil {
		return e.Enum
	}
	return nil
}

func (e *Tag) GetValueType() TagValueType {
	if e != nil {
		return e.ValueType
	}
	return TagValueTypeUnknown
}

func (e *Tag) GetStatus() TagStatus {
	if e != nil {
		return e.Status
	}
	return TagStatusUnknown
}

func (e *Tag) GetCreatorID() uint64 {
	if e != nil && e.CreatorID != nil {
		return *e.CreatorID
	}
	return 0
}

func (e *Tag) GetTenantID() uint64 {
	if e != nil && e.TenantID != nil {
		return *e.TenantID
	}
	return 0
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

func (e *Tag) IsValidTagValue(v string) bool {
	_, err := e.FormatTagValue(v)
	return err == nil
}

func (e *Tag) FormatTagValue(v string) (interface{}, error) {
	switch e.GetValueType() {
	case TagValueTypeStr:
		return fmt.Sprint(v), nil
	case TagValueTypeInt:
		if i, err := strconv.Atoi(v); err != nil {
			return nil, err
		} else {
			return i, nil
		}
	case TagValueTypeFloat:
		if f, err := strconv.ParseFloat(v, 64); err != nil {
			return nil, err
		} else {
			return floatFrac(f), nil
		}
	default:
		return nil, errors.New("unsupported tag value type")
	}
}
