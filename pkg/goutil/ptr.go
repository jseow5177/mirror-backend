package goutil

import (
	"reflect"
)

func String(s string) *string {
	return &s
}

func Uint32(ui uint32) *uint32 {
	return &ui
}

func Uint64(ui uint64) *uint64 {
	return &ui
}

func Float64(f float64) *float64 {
	return &f
}

func Int64(i int64) *int64 {
	return &i
}

func Int(i int) *int {
	return &i
}

func Bool(b bool) *bool {
	return &b
}

func AtLeastOne(args ...interface{}) bool {
	for _, arg := range args {
		if arg != nil && !reflect.ValueOf(arg).IsNil() {
			return true
		}
	}
	return false
}

func AtMostOne(args ...interface{}) bool {
	var noneNil int
	for _, arg := range args {
		if arg != nil && !reflect.ValueOf(arg).IsNil() {
			noneNil++
			if noneNil > 1 {
				return false
			}
		}
	}
	return true
}

func MustHaveOne(args ...interface{}) bool {
	var noneNil int
	for _, arg := range args {
		if arg != nil && !reflect.ValueOf(arg).IsNil() {
			noneNil++
			if noneNil > 1 {
				return false
			}
		}
	}
	return noneNil == 1
}

func IsNil(i interface{}) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Ptr, reflect.Map, reflect.Array, reflect.Chan, reflect.Slice:
		return reflect.ValueOf(i).IsNil()
	default:
		return false
	}
}

func Contains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}
