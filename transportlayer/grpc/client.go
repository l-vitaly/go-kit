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

func NewClient(serviceName string, conn *grpc.ClientConn, endpoints ...transportlayer.Endpoint) transportlayer.Client {
	methods := make(map[string]*grpctransport.Client)
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

		methods[m.Name()] = grpctransport.NewClient(
			conn,
			serviceName,
			m.Name(),
			converterGRPC.EncodeReq,
			converterGRPC.DecodeResp,
			converterGRPC.ReplyType,
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
