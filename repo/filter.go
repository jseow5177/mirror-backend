package repo

import (
	"cdp/pkg/goutil"
	"fmt"
)

type LogicalOp string

const (
	And LogicalOp = "AND"
	Or  LogicalOp = "OR"
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

type Condition struct {
	Field         string
	Op            Op
	Value         interface{}
	NextLogicalOp LogicalOp
}

func ToSqlWithArgs(conditions []*Condition) (sql string, args []interface{}) {
	for i, condition := range conditions {
		if goutil.IsNil(condition.Value) {
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

		if len(conditions) > 1 && i != len(conditions)-1 {
			sql += fmt.Sprintf(" %s ", condition.NextLogicalOp)
		}
	}

	return
}
