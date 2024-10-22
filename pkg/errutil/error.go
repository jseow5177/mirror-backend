package errutil

import (
	"errors"
	"net/http"
)

type HttpError struct {
	Code int
	Err  error
}

func (e HttpError) Error() string {
	return e.Err.Error()
}

func InternalServerError(err error) error {
	return HttpError{
		Code: http.StatusInternalServerError,
		Err:  err,
	}
}

func BadRequestError(err error) error {
	return HttpError{
		Code: http.StatusBadRequest,
		Err:  err,
	}
}

func ValidationError(err error) error {
	return HttpError{
		Code: http.StatusUnprocessableEntity,
		Err:  err,
	}
}

func NotFoundError(err error) error {
	return HttpError{
		Code: http.StatusNotFound,
		Err:  err,
	}
}

func ParseHttpError(err error) (int, string) {
	var httpErr HttpError
	if errors.As(err, &httpErr) {
		return httpErr.Code, httpErr.Error()
	}
	if err == nil {
		return http.StatusOK, ""
	}
	return http.StatusInternalServerError, err.Error()
}
