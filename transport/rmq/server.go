package rmq

import (
	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"golang.org/x/net/context"
)

// Handler which should be called from the RMQ RPC binding of the service
// implementation.
type Handler interface {
	ServeRMQ(rmqCtx context.Context, req interface{}) (context.Context, interface{}, error)
}

// Server wraps an endpoint and implements rmq.ServeRMQ.
type Server struct {
	ctx    context.Context
	e      endpoint.Endpoint
	dec    DecodeRequestFunc
	enc    EncodeResponseFunc
	before []RequestFunc
	after  []ResponseFunc
	logger log.Logger
}

// NewServer constructs a new server, which implements wraps the provided
// endpoint and implements the Handler interface.
func NewServer(
	ctx context.Context,
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	options ...ServerOption,
) *Server {
	s := &Server{
		ctx:    ctx,
		e:      e,
		dec:    dec,
		enc:    enc,
		logger: log.NewNopLogger(),
	}
	for _, option := range options {
		option(s)
	}
	return s
}

// ServerOption sets an optional parameter for servers.
type ServerOption func(*Server)

// ServerBefore functions are executed on the RMQ RPC request object before the
// request is decoded.
func ServerBefore(before ...RequestFunc) ServerOption {
	return func(s *Server) { s.before = before }
}

// ServerAfter functions are executed on the RMQ RPC response writer after the
// endpoint is invoked, but before anything is written to the client.
func ServerAfter(after ...ResponseFunc) ServerOption {
	return func(s *Server) { s.after = after }
}

// ServerErrorLogger is used to log non-terminal errors. By default, no errors
// are logged.
func ServerErrorLogger(logger log.Logger) ServerOption {
	return func(s *Server) { s.logger = logger }
}

// ServeRMQ implements the Handler interface.
func (s Server) ServeRMQ(rmqCtx context.Context, req interface{}) (context.Context, interface{}, error) {
	ctx := s.ctx

	for _, f := range s.before {
		ctx = f(ctx)
	}

	request, err := s.dec(rmqCtx, req)
	if err != nil {
		s.logger.Log("err", err)
		return rmqCtx, nil, BadRequestError{err}
	}

	response, err := s.e(ctx, request)
	if err != nil {
		s.logger.Log("err", err)
		return rmqCtx, nil, err
	}

	for _, f := range s.after {
		f(ctx)
	}

	rmqResp, err := s.enc(rmqCtx, response)
	if err != nil {
		s.logger.Log("err", err)
		return rmqCtx, nil, err
	}

	return rmqCtx, rmqResp, nil
}

// BadRequestError is an error in decoding the request.
type BadRequestError struct {
	Err error
}

// Error implements the error interface.
func (err BadRequestError) Error() string {
	return err.Err.Error()
}
