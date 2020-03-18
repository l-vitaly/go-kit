package jsonrpc

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/valyala/fasthttp"

	"github.com/pquerna/ffjson/ffjson"

	"github.com/go-kit/kit/log"
	fasthttptransport "github.com/l-vitaly/go-kit/transport/fasthttp"
)

// Server wraps an endpoint and implements http.Handler.
type Server struct {
	ecm          EndpointCodecMap
	before       []fasthttptransport.RequestFunc
	after        []fasthttptransport.ServerResponseFunc
	errorEncoder fasthttptransport.ErrorEncoder
	logger       log.Logger
}

// NewServer constructs a new server, which implements http.Server.
func NewServer(
	ecm EndpointCodecMap,
	options ...ServerOption,
) *Server {
	s := &Server{
		ecm:          ecm,
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
func ServerBefore(before ...fasthttptransport.RequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...fasthttptransport.ServerResponseFunc) ServerOption {
	return func(s *Server) { s.after = append(s.after, after...) }
}

// ServerErrorEncoder is used to encode errors to the http.ResponseWriter
// whenever they're encountered in the processing of a request. Clients can
// use this to provide custom error formatting and response codes. By default,
// errors will be written with the DefaultErrorEncoder.
func ServerErrorEncoder(ee fasthttptransport.ErrorEncoder) ServerOption {
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

// ServeHTTP implements http.Handler.
func (s Server) ServeFastHTTP(rctx *fasthttp.RequestCtx) {

	if string(rctx.Method()) != fasthttp.MethodGet {
		rctx.Response.Header.Set("Content-Type", "text/plain; charset=utf-8")
		rctx.SetStatusCode(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(rctx, "405 must POST\n")
		return
	}

	ctx := context.TODO()

	//if s.finalizer != nil {
	//	iw := &interceptingWriter{rctx, http.StatusOK}
	//	defer func() { s.finalizer(ctx, iw.code, r) }()
	//	rctx = iw
	//}

	for _, f := range s.before {
		ctx = f(ctx, &rctx.Request)
	}

	// Decode the body into an  object
	var req Request

	err := ffjson.Unmarshal(rctx.Request.Body(), &req)
	if err != nil {
		rpcerr := parseError("JSON could not be decoded: " + err.Error())
		_ = s.logger.Log("err", rpcerr)
		s.errorEncoder(ctx, rpcerr, rctx)
		return
	}

	// Get the endpoint and codecs from the map using the method
	// defined in the JSON  object
	ecm, ok := s.ecm[req.Method]
	if !ok {
		err := methodNotFoundError(fmt.Sprintf("Method %s was not found.", req.Method))
		_ = s.logger.Log("err", err)
		s.errorEncoder(ctx, err, rctx)
		return
	}

	// Decode the JSON "params"
	reqParams, err := ecm.Decode(ctx, req.Params)
	if err != nil {
		_ = s.logger.Log("err", err)
		s.errorEncoder(ctx, err, rctx)
		return
	}

	// Call the Endpoint with the params
	response, err := ecm.Endpoint(ctx, reqParams)
	if err != nil {
		_ = s.logger.Log("err", err)
		s.errorEncoder(ctx, err, rctx)
		return
	}

	for _, f := range s.after {
		ctx = f(ctx, &rctx.Response)
	}

	res := Response{
		ID:      req.ID,
		JSONRPC: Version,
	}

	// Encode the response from the Endpoint
	resParams, err := ecm.Encode(ctx, response)
	if err != nil {
		_ = s.logger.Log("err", err)
		s.errorEncoder(ctx, err, rctx)
		return
	}

	res.Result = resParams

	rctx.Response.Header.Set("Content-Type", ContentType)

	b, _ := ffjson.Marshal(res)

	_, _ = rctx.Write(b)
}

// DefaultErrorEncoder writes the error to the ResponseWriter,
// as a json-rpc error response, with an InternalError status code.
// The Error() string of the error will be used as the response error message.
// If the error implements ErrorCoder, the provided code will be set on the
// response error.
// If the error implements Headerer, the given headers will be set.
func DefaultErrorEncoder(_ context.Context, err error, rctx *fasthttp.RequestCtx) {
	rctx.Response.Header.Set("Content-Type", ContentType)
	if headerer, ok := err.(fasthttptransport.Headerer); ok {
		for k := range headerer.Headers() {
			rctx.Response.Header.Set(k, headerer.Headers()[k])
		}
	}

	e := Error{
		Code:    InternalError,
		Message: err.Error(),
	}
	if sc, ok := err.(ErrorCoder); ok {
		e.Code = sc.ErrorCode()
	}

	rctx.SetStatusCode(http.StatusOK)

	b, _ := ffjson.Marshal(Response{
		JSONRPC: Version,
		Error:   &e,
	})
	_, _ = rctx.Write(b)
}

// ErrorCoder is checked by DefaultErrorEncoder. If an error value implements
// ErrorCoder, the integer result of ErrorCode() will be used as the JSONRPC
// error code when encoding the error.
//
// By default, InternalError (-32603) is used.
type ErrorCoder interface {
	ErrorCode() int
}

// interceptingWriter intercepts calls to WriteHeader, so that a finalizer
// can be given the correct status code.
type interceptingWriter struct {
	fasthttp.RequestCtx
	code int
}

// WriteHeader may not be explicitly called, so care must be taken to
// initialize w.code to its default value of http.StatusOK.
func (w *interceptingWriter) WriteHeader(code int) {
	w.code = code
	w.RequestCtx.SetStatusCode(code)
}