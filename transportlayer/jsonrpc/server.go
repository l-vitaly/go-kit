package jsonrpc

import (
	"net/http"

	jsonrpctransport "github.com/l-vitaly/go-kit/transport/jsonrpc"
	"github.com/l-vitaly/go-kit/transportlayer"
)

type ServerOption func(*Server)

type Server struct {
	options map[string][]jsonrpctransport.ServerOption
	methods map[string]*jsonrpctransport.Server
}

func ServerJSONRPCOption(method string, o ...jsonrpctransport.ServerOption) ServerOption {
	return func(s *Server) {
		s.options[method] = append(s.options[method], o...)
	}
}

func NewServer(endpoints []transportlayer.Endpoint, options ...ServerOption) *Server {
	s := &Server{
		options: make(map[string][]jsonrpctransport.ServerOption),
		methods: make(map[string]*jsonrpctransport.Server),
	}

	for _, option := range options {
		option(s)
	}

	for _, m := range endpoints {
		var converterJSONRPC *EndpointServerConverter
		for _, converter := range m.Converters() {
			if c, ok := converter.(*EndpointServerConverter); ok {
				converterJSONRPC = c
				break
			}
		}

		if converterJSONRPC == nil {
			panic("JSONRPC converter not found")
		}

		var serverOptions []jsonrpctransport.ServerOption
		if options, ok := s.options[m.Name()]; ok {
			serverOptions = options
		}
		if globalOpts, ok := s.options["*"]; ok {
			serverOptions = append(serverOptions, globalOpts...)
		}

		s.methods[m.Name()] = jsonrpctransport.NewServer(
			m.Fn(),
			converterJSONRPC.DecodeReq,
			converterJSONRPC.EncodeResp,
			serverOptions...,
		)
	}
	return s
}

func (t *Server) Serve(r *http.Request, req interface{}) (interface{}, error) {
	methodName := transportlayer.GetCallerName()
	if srv, ok := t.methods[methodName]; ok {
		return srv.ServeJSONRPC(r, req)
	}
	return nil, ErrServerEndpointNotFound
}
