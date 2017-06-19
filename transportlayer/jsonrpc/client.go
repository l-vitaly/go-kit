package jsonrpc

import (
	"context"

	jsonrpctransport "github.com/l-vitaly/go-kit/transport/jsonrpc"
	"github.com/l-vitaly/go-kit/transportlayer"
)

type ClientOption func(*Client)

type Client struct {
	methods map[string]*jsonrpctransport.Client
}

func NewClient(url string, serviceName string, endpoints []transportlayer.Endpoint) *Client {
	c := &Client{
		methods: make(map[string]*jsonrpctransport.Client),
	}

	for _, m := range endpoints {
		var converterJSONRPC *EndpointClientConverter
		for _, converter := range m.Converters() {
			if c, ok := converter.(*EndpointClientConverter); ok {
				converterJSONRPC = c
				break
			}
		}

		if converterJSONRPC == nil {
			panic("GRPC converter not found")
		}

		c.methods[m.Name()] = jsonrpctransport.NewClient(
			url,
			serviceName,
			m.Name(),
			converterJSONRPC.EncodeReq,
			converterJSONRPC.DecodeResp,
			converterJSONRPC.ReplyType,
		)
	}
	return c
}

func (t *Client) Call(ctx context.Context, request interface{}) (response interface{}, err error) {
	methodName := transportlayer.GetCallerName()
	if e, ok := t.methods[methodName]; ok {
		return e.Endpoint()(ctx, request)
	}
	return ctx, ErrClientEndpointNotFound
}
