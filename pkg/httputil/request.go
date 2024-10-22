package httputil

import (
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
)

const (
	MaxFileSize = 10 << 20 // 10MB
)

func ReadJsonBody(r *http.Request, dst interface{}) error {
	if r.Body == http.NoBody {
		return nil
	}

	d := json.NewDecoder(r.Body)

	return d.Decode(dst)
}

func ParseFile(r *http.Request) ([]byte, error) {
	if err := r.ParseMultipartForm(MaxFileSize); err != nil {
		return nil, err
	}

	f, _, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer func(f multipart.File) {
		_ = f.Close()
	}(f)

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	return b, nil
}
