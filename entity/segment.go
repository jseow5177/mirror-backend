package entity

import (
	"encoding/json"
)

type LookupOp string

const (
	LookupOpEq  LookupOp = "="
	LookupOpGt  LookupOp = ">"
	LookupOpLt  LookupOp = "<"
	LookupOpGte LookupOp = ">="
	LookupOpLte LookupOp = "<="
	LookupOpIn  LookupOp = "in"
)

var SupportedLookupOps = []LookupOp{
	LookupOpEq,
	LookupOpGt,
	LookupOpLt,
	LookupOpGte,
	LookupOpLte,
	LookupOpIn,
}

type QueryOp string

const (
	QueryOpAnd QueryOp = "AND"
	QueryOpOr  QueryOp = "OR"
)

type SegmentStatus uint32

const (
	SegmentStatusUnknown SegmentStatus = iota
	SegmentStatusNormal
	SegmentStatusDeleted
)

type Lookup struct {
	TagID *uint64     `json:"tag_id,omitempty"`
	Op    LookupOp    `json:"op,omitempty"`
	Not   *bool       `json:"not,omitempty"`
	Val   interface{} `json:"val,omitempty"`
}

func (e *Lookup) GetTagID() uint64 {
	if e != nil && e.TagID != nil {
		return *e.TagID
	}
	return 0
}

func (e *Lookup) GetVal() interface{} {
	if e != nil && e.Val != nil {
		return e.Val
	}
	return nil
}

func (e *Lookup) GetNot() bool {
	if e != nil && e.Not != nil {
		return *e.Not
	}
	return false
}

type Query struct {
	Lookups []*Lookup `json:"lookups,omitempty"`
	Queries []*Query  `json:"queries,omitempty"`
	Op      QueryOp   `json:"op,omitempty"`
	Not     *bool     `json:"not,omitempty"`
}

func (e *Query) GetOp() QueryOp {
	if e != nil {
		return e.Op
	}
	return ""
}

func (e *Query) GetNot() bool {
	if e != nil && e.Not != nil {
		return *e.Not
	}
	return false
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
	ID          *uint64       `json:"id,omitempty"`
	Name        *string       `json:"name,omitempty"`
	SegmentDesc *string       `json:"segment_desc,omitempty"`
	Criteria    *Query        `json:"criteria,omitempty"`
	Status      SegmentStatus `json:"status,omitempty"`
	CreatorID   *uint64       `json:"creator_id,omitempty"`
	TenantID    *uint64       `json:"tenant_id,omitempty"`
	CreateTime  *uint64       `json:"create_time,omitempty"`
	UpdateTime  *uint64       `json:"update_time,omitempty"`
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

func (e *Segment) GetCriteria() *Query {
	if e != nil && e.Criteria != nil {
		return e.Criteria
	}
	return nil
}

func (e *Segment) GetSegmentDesc() string {
	if e != nil && e.SegmentDesc != nil {
		return *e.SegmentDesc
	}
	return ""
}

func (e *Segment) GetStatus() SegmentStatus {
	if e != nil {
		return e.Status
	}
	return SegmentStatusUnknown
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
