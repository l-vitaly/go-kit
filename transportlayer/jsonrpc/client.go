package jsonrpc

import (
	"context"

	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/l-vitaly/go-kit/transportlayer"
	"google.golang.org/grpc"
)

type ClientOption func(*Client)

type Client struct {
	options map[string][]grpctransport.ClientOption
	methods map[string]*grpctransport.Client
}

func ClientGRPCOption(method string, o ...grpctransport.ClientOption) ClientOption {
	return func(c *Client) {
		c.options[method] = append(c.options[method], o...)
	}
}

func NewClient(serviceName string, conn *grpc.ClientConn, endpoints []transportlayer.Endpoint, options ...ClientOption) *Client {
	c := &Client{
		options: make(map[string][]grpctransport.ClientOption),
		methods: make(map[string]*grpctransport.Client),
	}

	for _, option := range options {
		option(c)
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

		var clientOptions []grpctransport.ClientOption
		if options, ok := c.options[m.Name()]; ok {
			clientOptions = options
		}
		if globalOpts, ok := c.options["*"]; ok {
			clientOptions = append(clientOptions, globalOpts...)
		}

		c.methods[m.Name()] = grpctransport.NewClient(
			conn,
			serviceName,
			m.Name(),
			converterGRPC.EncodeReq,
			converterGRPC.DecodeResp,
			converterGRPC.ReplyType,
			clientOptions...,
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
