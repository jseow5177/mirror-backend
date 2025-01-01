package goutil

import (
	"encoding/base64"
	"errors"
	"golang.org/x/crypto/bcrypt"
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
	_, err := base64.StdEncoding.DecodeString(input)
	if err != nil {
		return errors.New("base64 decode error")
	}

	return nil
}

func BCrypt(s string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
