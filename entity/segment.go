package entity

import (
	"encoding/json"
)

const (
	QueryOpAnd = "AND"
	QueryOpOr  = "OR"
)

type SegmentStatus uint32

const (
	SegmentStatusUnknown SegmentStatus = iota
	SegmentStatusNormal
	SegmentStatusDeleted
)

type Range struct {
	Lte *string `json:"lte,omitempty"`
	Lt  *string `json:"lt,omitempty"`
	Gte *string `json:"gte,omitempty"`
	Gt  *string `json:"gt,omitempty"`
}

func (e *Range) GetLt() string {
	if e != nil && e.Lt != nil {
		return *e.Lt
	}
	return ""
}

func (e *Range) GetGt() string {
	if e != nil && e.Gt != nil {
		return *e.Gt
	}
	return ""
}

func (e *Range) GetGte() string {
	if e != nil && e.Gte != nil {
		return *e.Gte
	}
	return ""
}

func (e *Range) GetLte() string {
	if e != nil && e.Lte != nil {
		return *e.Lte
	}
	return ""
}

type Lookup struct {
	TagID *uint64  `json:"tag_id,omitempty"`
	Eq    *string  `json:"eq,omitempty"`
	Not   *bool    `json:"not,omitempty"`
	In    []string `json:"in,omitempty"`
	Range *Range   `json:"range,omitempty"`
}

func (e *Lookup) GetEq() string {
	if e != nil && e.Eq != nil {
		return *e.Eq
	}
	return ""
}

func (e *Lookup) GetTagID() uint64 {
	if e != nil && e.TagID != nil {
		return *e.TagID
	}
	return 0
}

type Query struct {
	Lookups []*Lookup `json:"lookups,omitempty"`
	Queries []*Query  `json:"queries,omitempty"`
	Op      *string   `json:"op,omitempty"`
	Not     *bool     `json:"not,omitempty"`
}

func (e *Query) GetOp() string {
	if e != nil && e.Op != nil {
		return *e.Op
	}
	return ""
}

func (e *Query) ToString() (string, error) {
	if e == nil {
		return "{}", nil
	}

	b, err := json.Marshal(e)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

type Segment struct {
	ID         *uint64 `json:"id,omitempty"`
	Name       *string `json:"name,omitempty"`
	Desc       *string `json:"desc,omitempty"`
	Query      *Query  `json:"query,omitempty"`
	Status     *uint32 `json:"status,omitempty"`
	CreateTime *uint64 `json:"create_time,omitempty"`
	UpdateTime *uint64 `json:"update_time,omitempty"`
}

func (e *Segment) GetID() uint64 {
	if e != nil && e.ID != nil {
		return *e.ID
	}
	return 0
}

func (e *Segment) GetName() string {
	if e != nil && e.Name != nil {
		return *e.Name
	}
	return ""
}

func (e *Segment) GetQuery() *Query {
	if e != nil && e.Query != nil {
		return e.Query
	}
	return nil
}

func (e *Segment) GetDesc() string {
	if e != nil && e.Desc != nil {
		return *e.Desc
	}
	return ""
}

func (e *Segment) GetStatus() uint32 {
	if e != nil && e.Status != nil {
		return *e.Status
	}
	return uint32(SegmentStatusUnknown)
}

func (e *Segment) GetCreateTime() uint64 {
	if e != nil && e.CreateTime != nil {
		return *e.CreateTime
	}
	return 0
}

func (e *Segment) GetUpdateTime() uint64 {
	if e != nil && e.UpdateTime != nil {
		return *e.UpdateTime
	}
	return 0
}
