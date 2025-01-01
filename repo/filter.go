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

type Filter struct {
	Conditions []*Condition
	Pagination *Pagination
}

type Condition struct {
	Field         string
	Op            Op
	Value         interface{}
	NextLogicalOp LogicalOp
}

func ToSqlWithArgs(f *Filter) (sql string, args []interface{}) {
	for i, condition := range f.Conditions {
		if condition == nil || goutil.IsNil(condition.Value) {
			continue
		}

		switch condition.Op {
		case OpEq:
			sql += fmt.Sprintf("%s = ?", condition.Field)
			args = append(args, condition.Value)
		case OpNotEq:
			sql += fmt.Sprintf("%s != ?", condition.Field)
			args = append(args, condition.Value)
		case OpGt:
			sql += fmt.Sprintf("%s > ?", condition.Field)
			args = append(args, condition.Value)
		case OpGte:
			sql += fmt.Sprintf("%s >= ?", condition.Field)
			args = append(args, condition.Value)
		case OpLt:
			sql += fmt.Sprintf("%s < ?", condition.Field)
			args = append(args, condition.Value)
		case OpLte:
			sql += fmt.Sprintf("%s <= ?", condition.Field)
			args = append(args, condition.Value)
		case OpLike:
			sql += fmt.Sprintf("%s LIKE ?", condition.Field)
			args = append(args, condition.Value)
		case OpIn:
			sql += fmt.Sprintf("%s IN ?", condition.Field)
			args = append(args, condition.Value)
		}

		logicalOp := condition.NextLogicalOp
		if logicalOp == "" {
			logicalOp = LogicalOpAnd
		}

		if len(f.Conditions) > 1 && i != len(f.Conditions)-1 {
			sql += fmt.Sprintf(" %s ", logicalOp)
		}
	}

	return
}
