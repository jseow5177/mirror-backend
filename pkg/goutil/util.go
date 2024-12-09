package goutil

import (
	"encoding/base64"
	"errors"
	"strings"
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

func IsBase64EncodedHTML(input string) error {
	decoded, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return errors.New("base64 decode error")
	}

	decodedStr := string(decoded)

	if !strings.Contains(decodedStr, "<html>") && !strings.Contains(decodedStr, "<body>") {
		return errors.New("invalid html")
	}

	return nil
}
