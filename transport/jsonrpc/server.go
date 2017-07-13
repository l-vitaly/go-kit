package jsonrpc

import (
    "context"

    "github.com/go-kit/kit/endpoint"
    "github.com/go-kit/kit/log"
)

type Handler interface {
    ServeJSONRPC(ctx context.Context, req interface{}) (interface{}, error)
}

// Server wraps an endpoint and implements http.Handler.
type Server struct {
    e      endpoint.Endpoint
    dec    DecodeRequestFunc
    enc    EncodeResponseFunc
    before []ServerRequestFunc
    logger log.Logger
}

func NewServer(
    e endpoint.Endpoint,
    dec DecodeRequestFunc,
    enc EncodeResponseFunc,
    options ...ServerOption,
) *Server {
    s := &Server{
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

// ServeHTTP implements http.Handler.
func (s Server) ServeJSONRPC(ctx context.Context, req interface{}) (interface{}, error) {
    for _, f := range s.before {
        ctx = f(ctx)
    }
    request, err := s.dec(ctx, req)
    if err != nil {
        s.logger.Log("err", err)
        return nil, err
    }
    response, err := s.e(ctx, request)
    if err != nil {
        s.logger.Log("err", err)
        return nil, err
    }
    jrpcResp, err := s.enc(ctx, response)
    if err != nil {
        s.logger.Log("err", err)
        return nil, err
    }
    return jrpcResp, nil
}
