package grpc

import (
	"context"
	"log"

	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/l-vitaly/go-kit/transportlayer"
)

type serverGRPC struct {
	methods map[string]*grpctransport.Server
}

func NewServer(endpoints transportlayer.Endpoints) transportlayer.Server {
	methods := make(map[string]*grpctransport.Server)

	for _, m := range endpoints.Endpoints() {
		methods[m.Name()] = grpctransport.NewServer(
			m.Fn(),
			m.Decode().Request().(grpctransport.DecodeRequestFunc),
			m.Encode().Response().(grpctransport.EncodeResponseFunc),
		)
	}
	return &serverGRPC{methods: methods}
}

func (t *serverGRPC) Serve(ctx context.Context, req interface{}) (context.Context, interface{}, error) {
	methodName := transportlayer.GetCallerName()
	log.Println(methodName)
	if srv, ok := t.methods[methodName]; ok {
		return srv.ServeGRPC(ctx, req)
	}
	return ctx, nil, ErrEndpointNotFound
}
