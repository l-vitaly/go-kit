package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/go-kit/kit/log"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gorilla/websocket"
)

type requestIDKeyType struct{}

var RequestIDKey requestIDKeyType

// Server wraps an endpoint and implements http.Handler.
type Server struct {
	upgrader     websocket.Upgrader
	ecm          EndpointCodecMap
	before       []httptransport.RequestFunc
	after        []httptransport.ServerResponseFunc
	errorEncoder ErrorEncoder
	finalizer    httptransport.ServerFinalizerFunc
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
func ServerBefore(before ...httptransport.RequestFunc) ServerOption {
	return func(s *Server) { s.before = append(s.before, before...) }
}

// ServerAfter functions are executed on the HTTP response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...httptransport.ServerResponseFunc) ServerOption {
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

// ErrorEncoder is responsible for encoding an error to the ResponseWriter.
// Users are encouraged to use custom ErrorEncoders to encode HTTP errors to
// their clients, and will likely want to pass and check for their own error
// types. See the example shipping/handling service.
type ErrorEncoder func(ctx context.Context, err error) Response

// ResponseWriter ...
type ResponseWriter func(ctx context.Context, responses []Response, isBatch bool, w http.ResponseWriter)

// ServerFinalizer is executed at the end of every HTTP request.
// By default, no finalizer is registered.
func ServerFinalizer(f httptransport.ServerFinalizerFunc) ServerOption {
	return func(s *Server) { s.finalizer = f }
}

// ServeHTTP implements http.Handler.
func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.finalizer != nil {
		iw := &interceptingWriter{w, http.StatusOK}
		defer func() { s.finalizer(ctx, iw.code, r) }()
		w = iw
	}
	for _, f := range s.before {
		ctx = f(ctx, r)
	}
	s.serveHTTP(ctx, w, r)
}

func (s Server) marshalResponse(responses []Response, isBatch bool) (data []byte) {
	if !isBatch && len(responses) > 0 {
		data, _ = json.Marshal(responses[0])
		return
	}
	data, _ = json.Marshal(responses)
	return
}

// ServeHTTP implements http.Handler.
func (s Server) serveHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", ContentType)

	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, _ = io.WriteString(w, "405 must POST\n")
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		rpcerr := parseError("JSON could not be read body: " + err.Error())
		_ = s.logger.Log("err", rpcerr)
		_, _ = w.Write(s.marshalResponse([]Response{s.errorEncoder(ctx, rpcerr)}, false))
		return
	}

	async := r.Header.Get("X-Async") == "on"

	result, isBatch, err := s.rpcCall(ctx, body, async)
	if err != nil {
		rpcerr := parseError("jsonrpc internal error: " + err.Error())
		_ = s.logger.Log("err", rpcerr)
		_, _ = w.Write(s.marshalResponse([]Response{s.errorEncoder(ctx, rpcerr)}, false))
		return
	}

	for _, f := range s.after {
		ctx = f(ctx, w)
	}

	_, _ = w.Write(s.marshalResponse(result, isBatch))
}

func (s Server) rpcCall(ctx context.Context, data []byte, async bool) (result []Response, isBatch bool, err error) {
	isBatch = true
	if !bytes.HasPrefix(data, []byte("[")) && !bytes.HasSuffix(data, []byte("]")) {
		isBatch = false
		buf := new(bytes.Buffer)
		buf.WriteString("[")
		buf.Write(data)
		buf.WriteString("]")
		data = buf.Bytes()
	}

	// Decode the body into an object
	var reqs []Request
	err = json.Unmarshal(data, &reqs)
	if err != nil {
		return nil, false, err
	}

	responses := make(chan Response, len(reqs))

	for _, req := range reqs {
		ctx = context.WithValue(ctx, RequestIDKey, req.ID)
		// Get the endpoint and codecs from the map using the method
		// defined in the JSON  object
		ecm, ok := s.ecm[req.Method]
		if !ok {
			err := methodNotFoundError(fmt.Sprintf("Method %s was not found.", req.Method))
			_ = s.logger.Log("err", err)
			responses <- s.errorEncoder(ctx, err)
			continue
		}

		// Decode the JSON "params"
		reqParams, err := ecm.Decode(ctx, req.Params)
		if err != nil {
			_ = s.logger.Log("err", err)
			responses <- s.errorEncoder(ctx, err)
			continue
		}

		reqFn := func(ctx context.Context, req Request, reqParams interface{}) {
			// Call the Endpoint with the params
			response, err := ecm.Endpoint(ctx, reqParams)
			if err != nil {
				_ = s.logger.Log("err", err)
				responses <- s.errorEncoder(ctx, err)
				return
			}

			res := Response{
				ID:      req.ID,
				JSONRPC: Version,
			}

			// Encode the response from the Endpoint
			resParams, err := ecm.Encode(ctx, response)
			if err != nil {
				_ = s.logger.Log("err", err)
				responses <- s.errorEncoder(ctx, err)
				return
			}
			res.Result = resParams
			responses <- res
		}
		if async {
			go reqFn(ctx, req, reqParams)
		} else {
			reqFn(ctx, req, reqParams)
		}
	}

	for i := 0; i < len(reqs); i++ {
		result = append(result, <-responses)
	}
	return
}

// DefaultErrorEncoder writes the error to the ResponseWriter,
// as a json-rpc error response, with an InternalError status code.
// The Error() string of the error will be used as the response error message.
// If the error implements ErrorCoder, the provided code will be set on the
// response error.
// If the error implements Headerer, the given headers will be set.
func DefaultErrorEncoder(ctx context.Context, err error) Response {
	e := Error{
		Code:    InternalError,
		Message: err.Error(),
	}
	if sc, ok := err.(ErrorCoder); ok {
		e.Code = sc.ErrorCode()
	}

	if sc, ok := err.(ErrorData); ok {
		e.Data = sc.ErrorData()
	}

	var requestID *RequestID
	if v := ctx.Value(RequestIDKey); v != nil {
		requestID = v.(*RequestID)
	}

	return Response{
		ID:      requestID,
		JSONRPC: Version,
		Error:   &e,
	}
}

// ErrorCoder is checked by DefaultErrorEncoder. If an error value implements
// ErrorCoder, the integer result of ErrorCode() will be used as the JSONRPC
// error code when encoding the error.
//
// By default, InternalError (-32603) is used.
type ErrorCoder interface {
	ErrorCode() int
}

// ErrorData is checked by DefaultErrorEncoder. If an error value implements
// ErrorData, the interface{} result of ErrorData() will be used as the JSONRPC
// error data when encoding the error.
//
// By default, empty is used.
type ErrorData interface {
	ErrorData() interface{}
}

// interceptingWriter intercepts calls to WriteHeader, so that a finalizer
// can be given the correct status code.
type interceptingWriter struct {
	http.ResponseWriter
	code int
}

// WriteHeader may not be explicitly called, so care must be taken to
// initialize w.code to its default value of http.StatusOK.
func (w *interceptingWriter) WriteHeader(code int) {
	w.code = code
	w.ResponseWriter.WriteHeader(code)
}
