package router

import (
	"cdp/pkg/errutil"
	"cdp/pkg/httputil"
	"context"
	"errors"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"mime/multipart"
	"net/http"
	"reflect"
)

type FileInfo struct {
	File       multipart.File
	FileHeader *multipart.FileHeader
}

// to decode url params
var decoder = schema.NewDecoder()

var (
	ErrUnsupportedContentType = errors.New("unsupported content type")
	ErrCannotSetFileInfo      = errors.New("cannot set file info")
	ErrCannotDecodeUrlParams  = errors.New("cannot decode url params")
)

type ContentType uint32

const (
	ContentTypeJSON ContentType = iota
	ContentTypeFile
)

type Middleware interface {
	Handle(http.Handler) http.Handler
}

type Handler struct {
	Req         interface{}
	Res         interface{}
	HandleFunc  func(ctx context.Context, req interface{}, res interface{}) error
	ContentType ContentType

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

	// decode url query
	if err := decoder.Decode(req, r.URL.Query()); err != nil {
		httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(ErrCannotDecodeUrlParams))
		return
	}

	switch h.ContentType {
	case ContentTypeJSON:
		if err := httputil.ReadJsonBody(r, req); err != nil {
			httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(err))
			return
		}
	case ContentTypeFile:
		f, fh, err := r.FormFile("file")
		if err != nil && !errors.Is(err, http.ErrMissingFile) {
			httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(err))
			return
		}
		defer func(f multipart.File) {
			if f != nil {
				_ = f.Close()
			}
		}(f)

		var (
			fileMeta = &FileInfo{
				File:       f,
				FileHeader: fh,
			}
			reqVal  = reflect.ValueOf(req).Elem()
			reqType = reqVal.Type()
		)
		if fileMetaField, ok := reqType.FieldByName("FileInfo"); ok {
			fv := reqVal.FieldByName(fileMetaField.Name)
			if fv.CanSet() {
				fv.Set(reflect.ValueOf(fileMeta)) // set to FileInfo field
			} else {
				httputil.ReturnServerResponse(w, nil, ErrCannotSetFileInfo)
				return
			}
		}
	default:
		httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(ErrUnsupportedContentType))
		return
	}

	err := h.HandleFunc(r.Context(), req, res)
	httputil.ReturnServerResponse(w, res, err)
}
