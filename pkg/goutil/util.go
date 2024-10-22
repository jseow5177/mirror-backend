package goutil

import (
	"fmt"
	"strconv"
)

func ContainsStr(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}

func ContainsUint32(arr []uint32, i uint32) bool {
	for _, v := range arr {
		if v == i {
			return true
		}
	}
	return false
}

func FormatFloat(s string, dp uint32) (string, error) {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return "", err
	}

	format := fmt.Sprintf("%%.%df", dp)
	return fmt.Sprintf(format, f), nil
}
