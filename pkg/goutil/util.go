package goutil

import (
	cryptoRand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"net/url"
	"reflect"
	"sort"
)

const characters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GenerateRandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = characters[rand.Intn(len(characters))]
	}
	return string(b)
}

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
	_, err := Base64Decode(input)
	if err != nil {
		return errors.New("base64 decode error")
	}

	return nil
}

func Sha256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func BCrypt(s string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CompareBCrypt(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

func generateSecureRandBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := cryptoRand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func GenerateSecureRandString(n int) (string, error) {
	b, err := generateSecureRandBytes(n)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func Base64Decode(s string) (string, error) {
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func BuildURL(domain, path string, params map[string]string) (string, error) {
	u, err := url.Parse(domain)
	if err != nil {
		return "", err
	}

	u.Path = path

	query := u.Query()
	for key, value := range params {
		query.Set(key, value)
	}
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func RemoveStrDuplicates(elems []string) []string {
	var (
		m      = make(map[string]bool)
		result = make([]string, 0)
	)
	for _, v := range elems {
		if !m[v] {
			m[v] = true
			result = append(result, v)
		}
	}

	return result
}

func IsStrArrEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sortedA := append([]string{}, a...)
	sortedB := append([]string{}, b...)
	sort.Strings(sortedA)
	sort.Strings(sortedB)

	return reflect.DeepEqual(sortedA, sortedB)
}
