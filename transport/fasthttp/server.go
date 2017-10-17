package fasthttp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

// Server wraps an endpoint and implements http.Handler.
type Server struct {
	e            endpoint.Endpoint
	dec          DecodeRequestFunc
	enc          EncodeResponseFunc
	before       []ServerRequestFunc
	after        []ServerResponseFunc
	errorEncoder ErrorEncoder
	logger       log.Logger
}

// NewServer constructs a new server, which implements http.Handler and wraps
// the provided endpoint.
func NewServer(
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	options ...ServerOption,
) *Server {
	s := &Server{
		e:            e,
		dec:          dec,
		enc:          enc,
		errorEncoder: DefaultErrorEncoder,
		logger:       log.NewNopLogger(),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// ServerOption sets an optional parameter for servers.
type ServerOption func(*Server)

// ServerBefore functions are executed on the HTTP request object before the
// request is decoded.
func ServerBefore(before ...ServerRequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...ServerResponseFunc) ServerOption {
	return func(s *Server) { s.after = append(s.after, after...) }
}

// ServerErrorEncoder is used to encode errors to the http.ResponseWriter
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting and response codes. By default,
// errors will be written with the DefaultErrorEncoder.
func ServerErrorEncoder(ee ErrorEncoder) ServerOption {
	return func(s *Server) { s.errorEncoder = ee }
}

// ServerErrorLogger is used to log non-terminal errors. By default, no errors
// are logged. This is intended as a diagnostic measure. Finer-grained control
// of error handling, including logging in more detail, should be performed in a
// custom ServerErrorEncoder or ServerFinalizer, both of which have access to
// the context.
func ServerErrorLogger(logger log.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

func (s Server) RouterHandle() routing.Handler {
	return func(ctx *routing.Context) error {
		s.Handle(ctx.RequestCtx)
		return nil
	}
}

// HandleFastHTTP implements fasthttp.HandleFastHTTP.
func (s Server) Handle(rctx *fasthttp.RequestCtx) {
	ctx := context.TODO()

	for _, f := range s.before {
		ctx = f(ctx, rctx)
	}

	request, err := s.dec(ctx, &rctx.Request)
	if err != nil {
		s.logger.Log("err", err)
		s.errorEncoder(ctx, err, &rctx.Response)
		return
	}

	response, err := s.e(ctx, request)
	if err != nil {
		s.logger.Log("err", err)
		s.errorEncoder(ctx, err, &rctx.Response)
		return
	}

	for _, f := range s.after {
		ctx = f(ctx, &rctx.Response)
	}

	if err := s.enc(ctx, &rctx.Response, response); err != nil {
		s.logger.Log("err", err)
		s.errorEncoder(ctx, err, &rctx.Response)
		return
	}
}

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error, rctx *fasthttp.Response)

// EncodeJSONResponse is a EncodeResponseFunc that serializes the response as a
// JSON object to the ResponseWriter. Many JSON-over-HTTP services can use it as
// a sensible default. If the response implements Headerer, the provided headers
// will be applied to the response. If the response implements StatusCoder, the
// provided StatusCode will be used instead of 200.
func EncodeJSONResponse(_ context.Context, r *fasthttp.Response, response interface{}) error {
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	if headerer, ok := response.(Headerer); ok {
		for k, v := range headerer.Headers() {
			r.Header.Set(k, v)
		}
	}
	code := http.StatusOK
	if sc, ok := response.(StatusCoder); ok {
		code = sc.StatusCode()
	}
	r.SetStatusCode(code)
	if code == http.StatusNoContent {
		return nil
	}
	b, err := json.Marshal(response)
	if err != nil {
		return err
	}
	r.SetBody(b)
	return nil
}

// DefaultErrorEncoder writes the error to the ResponseWriter, by default a
// content type of text/plain, a body of the plain text of the error, and a
// status code of 500. If the error implements Headerer, the provided headers
// will be applied to the response. If the error implements json.Marshaler, and
// the marshaling succeeds, a content type of application/json and the JSON
// encoded form of the error will be used. If the error implements StatusCoder,
// the provided StatusCode will be used instead of 500.
func DefaultErrorEncoder(_ context.Context, err error, r *fasthttp.Response) {
	contentType, body := "text/plain; charset=utf-8", []byte(err.Error())
	if marshaler, ok := err.(json.Marshaler); ok {
		if jsonBody, marshalErr := marshaler.MarshalJSON(); marshalErr == nil {
			contentType, body = "application/json; charset=utf-8", jsonBody
		}
	}
	r.Header.Set("Content-Type", contentType)
	if headerer, ok := err.(Headerer); ok {
		for k, v := range headerer.Headers() {
			r.Header.Set(k, v)
		}
	}
	code := http.StatusInternalServerError
	if sc, ok := err.(StatusCoder); ok {
		code = sc.StatusCode()
	}
	r.SetStatusCode(code)
	r.SetBody(body)
}

// StatusCoder is checked by DefaultErrorEncoder. If an error value implements
// StatusCoder, the StatusCode will be used when encoding the error. By default,
// StatusInternalServerError (500) is used.
type StatusCoder interface {
	StatusCode() int
}

// Headerer is checked by DefaultErrorEncoder. If an error value implements
// Headerer, the provided headers will be applied to the response writer, after
// the Content-Type is set.
type Headerer interface {
	Headers() map[string]string
}
