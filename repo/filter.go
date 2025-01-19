package repo

import (
	"cdp/pkg/goutil"
	"fmt"
)

type LogicalOp string

const (
	LogicalOpAnd LogicalOp = "AND"
	LogicalOpOr  LogicalOp = "OR"
)

type Op string

const (
	OpEq    Op = "="
	OpNotEq Op = "!="
	OpGt    Op = ">"
	OpGte   Op = ">="
	OpLt    Op = "<"
	OpLte   Op = "<="
	OpLike  Op = "LIKE"
	OpIn    Op = "IN"
)

type Pagination struct {
	Limit   *uint32 `json:"limit,omitempty"`
	Page    *uint32 `json:"page,omitempty"`
	Total   *uint32 `json:"total,omitempty"`
	HasNext *bool   `json:"has_next,omitempty"`
}

func (p *Pagination) GetLimit() uint32 {
	if p != nil && p.Limit != nil {
		return *p.Limit
	}
	return 0
}

func (p *Pagination) GetPage() uint32 {
	if p != nil && p.Page != nil {
		return *p.Page
	}
	return 0
}

func (p *Pagination) GetTotal() uint32 {
	if p != nil && p.Total != nil {
		return *p.Total
	}
	return 0
}

type Filter struct {
	Conditions []*Condition
	Pagination *Pagination
}

type Condition struct {
	Field         string
	Op            Op
	Value         interface{}
	NextLogicalOp LogicalOp
	OpenBracket   bool
	CloseBracket  bool
}

func ToSqlWithArgs(f *Filter) (sql string, args []interface{}) {
	for i, condition := range f.Conditions {
		if condition == nil || goutil.IsNil(condition.Value) {
			continue
		}

		var subSql string
		if condition.OpenBracket {
			subSql += "("
		}

		subSql += fmt.Sprintf("%s %s ?", condition.Field, condition.Op)

		if condition.CloseBracket {
			subSql += ")"
		}

		logicalOp := condition.NextLogicalOp
		if logicalOp == "" {
			logicalOp = LogicalOpAnd
		}

		if len(f.Conditions) > 1 && i != len(f.Conditions)-1 {
			subSql += fmt.Sprintf(" %s ", logicalOp)
		}

		sql += subSql
		args = append(args, condition.Value)
	}

	return
}
