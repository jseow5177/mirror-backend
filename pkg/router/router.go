package router

import (
	"cdp/pkg/errutil"
	"cdp/pkg/httputil"
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"mime"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
)

type FileMeta struct {
	File       multipart.File
	FileHeader *multipart.FileHeader
}

// to decode url params
var decoder = schema.NewDecoder()

var (
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrUnsupportedHttpMethod  = errors.New("unsupported http method")
	ErrCannotSetFileInfo      = errors.New("cannot set file info")
	ErrCannotDecodeUrlParams  = errors.New("cannot decode url params")
)

type Middleware interface {
	Handle(http.Handler) http.Handler
}

type Handler struct {
	Req        interface{}
	Res        interface{}
	HandleFunc func(ctx context.Context, req interface{}, res interface{}) error

	reqT  reflect.Type
	respT reflect.Type
}

type HttpRoute struct {
	Method      string
	Path        string
	Handler     Handler
	Middlewares []Middleware
}

type HttpRouter struct {
	*mux.Router
}

func (r *HttpRouter) RegisterHttpRoute(hr *HttpRoute) {
	// save req and res type
	hr.Handler.reqT = reflect.TypeOf(hr.Handler.Req).Elem()
	hr.Handler.respT = reflect.TypeOf(hr.Handler.Res).Elem()

	// calling chain
	chain := http.Handler(hr.Handler)

	if hr.Middlewares != nil {
		// wrap middlewares from right to left
		for i := len(hr.Middlewares) - 1; i >= 0; i-- {
			chain = hr.Middlewares[i].Handle(chain)
		}
	}

	r.Methods(hr.Method).Path(hr.Path).Handler(chain)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	req := reflect.New(h.reqT).Interface()
	res := reflect.New(h.respT).Interface()

	if err := decoder.Decode(req, r.URL.Query()); err != nil {
		httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(ErrCannotDecodeUrlParams))
		return
	}

	if hasContentType(r, "application/json") {
		if err := httputil.ReadJsonBody(r, req); err != nil {
			httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(err))
			return
		}
	} else if hasContentType(r, "multipart/form-data") {
		fileMeta, err := getFileMeta(r)
		if err != nil {
			httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(err))
			return
		}

		// set to FileMeta field in request struct
		reqVal := reflect.ValueOf(req).Elem()
		if fileMetaField, ok := reqVal.Type().FieldByName("FileMeta"); ok {
			fv := reqVal.FieldByName(fileMetaField.Name)
			if fv.CanSet() {
				fv.Set(reflect.ValueOf(fileMeta))
			} else {
				httputil.ReturnServerResponse(w, nil, ErrCannotSetFileInfo)
				return
			}
		}
	} else {
		httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(ErrUnsupportedContentType))
		return
	}

	err := h.HandleFunc(r.Context(), req, res)
	httputil.ReturnServerResponse(w, res, err)

	return
}

func getFileMeta(r *http.Request) (*FileMeta, error) {
	f, fh, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	fileMeta := &FileMeta{
		File:       f,
		FileHeader: fh,
	}

	return fileMeta, nil
}

func hasContentType(r *http.Request, mimetype string) bool {
	contentType := r.Header.Get("Content-type")
	if contentType == "" {
		return mimetype == "application/octet-stream"
	}

	for _, v := range strings.Split(contentType, ",") {
		t, _, err := mime.ParseMediaType(v)
		if err != nil {
			break
		}
		if t == mimetype {
			return true
		}
	}
	return false
}
