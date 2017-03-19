package grpc

import (
	"context"

	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/l-vitaly/go-kit/transportlayer"
)

type ServerOption func(*serverGRPC)

type serverGRPC struct {
    options map[string][]grpctransport.ServerOption
	methods map[string]*grpctransport.Server
}

func ServerGRPCOption(method string, o ...grpctransport.ServerOption) ServerOption {
    return func(s *serverGRPC) {
        s.options[method] = append(s.options[method], o...)
    }
}

func NewServer(endpoints []transportlayer.Endpoint, options ...ServerOption) transportlayer.Server {
    s := &serverGRPC{
        options: make(map[string][]grpctransport.ServerOption),
        methods: make(map[string]*grpctransport.Server),
    }

    for _, option := range options {
        option(s)
    }

	for _, m := range endpoints {
		var converterGRPC *EndpointConverter
		for _, converter := range m.Converters() {
			if c, ok := converter.(*EndpointConverter); ok {
				converterGRPC = c
				break
			}
		}

		if converterGRPC == nil {
			panic("GRPC converter not found")
		}

        var serverOptions []grpctransport.ServerOption
        if options, ok := s.options[m.Name()]; ok {
            serverOptions = options
        }
        if globalOpts, ok := s.options["*"]; ok {
            serverOptions = append(serverOptions, globalOpts...)
        }

        s.methods[m.Name()] = grpctransport.NewServer(
			m.Fn(),
			converterGRPC.DecodeReq,
			converterGRPC.EncodeResp,
			serverOptions...,
		)
	}
	return s
}

func (t *serverGRPC) Serve(ctx context.Context, req interface{}) (context.Context, interface{}, error) {
	methodName := transportlayer.GetCallerName()
	if srv, ok := t.methods[methodName]; ok {
		return srv.ServeGRPC(ctx, req)
	}
	return ctx, nil, ErrEndpointNotFound
}
