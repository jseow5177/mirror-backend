package handler

import (
	"cdp/pkg/goutil"
	"cdp/pkg/router"
	"cdp/pkg/validator"
	"errors"
	"regexp"
)

var (
	ErrMissingFile      = errors.New("missing file")
	ErrFileSizeTooLarge = errors.New("file size too large")
	ErrInvalidFileType  = errors.New("invalid file type")
)

func TagNameValidator(optional bool) validator.Validator {
	return &validator.String{
		Optional:  optional,
		UnsetZero: true,
		MinLen:    2,
		MaxLen:    60,
		Regex:     regexp.MustCompile(`^[0-9a-zA-Z_.\s]+$`),
	}
}

type fileInfoValidator struct {
	maxSize     int64
	contentType []string
	optional    bool
}

func (v *fileInfoValidator) Validate(value interface{}) error {
	fileInfo, ok := value.(*router.FileMeta)
	if !ok {
		return errors.New("expect FileInfo")
	}

	if fileInfo == nil || fileInfo.File == nil {
		if !v.optional {
			return ErrMissingFile
		}
	} else {
		if fileInfo.FileHeader.Size > v.maxSize {
			return ErrFileSizeTooLarge
		}
		if len(v.contentType) > 0 && !goutil.ContainsStr(v.contentType, fileInfo.FileHeader.Header.Get("Content-Type")) {
			return ErrInvalidFileType
		}
	}

	return nil
}

func FileInfoValidator(optional bool, maxSize int64, contentType []string) validator.Validator {
	return &fileInfoValidator{
		optional:    optional,
		maxSize:     maxSize,
		contentType: contentType,
	}
}
