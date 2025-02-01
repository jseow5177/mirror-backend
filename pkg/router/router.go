package router

import (
	"cdp/entity"
	"cdp/pkg/errutil"
	"cdp/pkg/httputil"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/rs/zerolog/log"
	"mime"
	"net/http"
	"reflect"
	"strings"
)

const (
	appBasePath = "/api/v1"
)

type FileUpload interface {
	SetFileMeta(m *entity.FileMeta)
}

// to decode url params
var decoder = schema.NewDecoder()

var (
	ErrUnsupportedContentType = errors.New("unsupported content type")
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

	r.Methods(hr.Method).Path(fmt.Sprintf("%s%s", appBasePath, hr.Path)).Handler(chain)
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var (
		ctx = r.Context()
		req = reflect.New(h.reqT).Interface()
		res = reflect.New(h.respT).Interface()
	)

	if err := decoder.Decode(req, r.URL.Query()); err != nil {
		log.Ctx(ctx).Error().Msgf("decode url query params error: %v", err)
		httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(ErrCannotDecodeUrlParams))
		return
	}

	if r.Body != http.NoBody {
		if hasContentType(r, "application/json") {
			if err := httputil.ReadJsonBody(r, req); err != nil {
				log.Ctx(ctx).Error().Msgf("read json body error: %v", err)
				httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(err))
				return
			}
		} else if hasContentType(r, "multipart/form-data") {
			fileMeta, err := getFileMeta(r)
			if err != nil {
				log.Ctx(ctx).Error().Msgf("get file meta error: %v", err)
				httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(err))
				return
			}

			if fileUpload, ok := req.(FileUpload); ok {
				fileUpload.SetFileMeta(fileMeta)
			}
		} else {
			httputil.ReturnServerResponse(w, nil, errutil.BadRequestError(ErrUnsupportedContentType))
			return
		}
	}

	if contextInfo, ok := req.(ContextInfo); ok {
		var (
			user, userOk     = GetUserFromContext(ctx)
			tenant, tenantOk = GetTenantFromContext(ctx)
		)

		if userOk {
			contextInfo.SetUser(user)
		}
		if tenantOk {
			contextInfo.SetTenant(tenant)
		}
	}

	err := h.HandleFunc(ctx, req, res)
	httputil.ReturnServerResponse(w, res, err)

	return
}

func getFileMeta(r *http.Request) (*entity.FileMeta, error) {
	f, fh, err := r.FormFile("file")
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()

	fileMeta := &entity.FileMeta{
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
