package grpc

import (
	"context"

	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/l-vitaly/go-kit/transportlayer"
	"google.golang.org/grpc"
)

type clientGRPC struct {
	methods map[string]*grpctransport.Client
}

func NewClient(serviceName string, t transportlayer.Endpoints, conn *grpc.ClientConn) transportlayer.Client {
	methods := make(map[string]*grpctransport.Client)
	for _, m := range t.Endpoints() {
		methods[m.Name()] = grpctransport.NewClient(
			conn,
			serviceName,
			m.Name(),
			m.Encode().Request().(grpctransport.EncodeRequestFunc),
			m.Decode().Response().(grpctransport.DecodeResponseFunc),
			m.Reply(),
		)
	}
	return &clientGRPC{methods: methods}
}

func (t *clientGRPC) Call(ctx context.Context, request interface{}) (response interface{}, err error) {
	methodName := transportlayer.GetCallerName()
	if e, ok := t.methods[methodName]; ok {
		return e.Endpoint()(ctx, request)
	}
	return ctx, ErrEndpointNotFound
}
