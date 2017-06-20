package jsonrpc

import (
    "context"

    jsonrpctransport "github.com/l-vitaly/go-kit/transport/jsonrpc"
    "github.com/l-vitaly/go-kit/transportlayer"
)

type ClientOption func(*Client)

func JSONRPCOption(method string, o ...jsonrpctransport.ClientOption) ClientOption {
    return func(c *Client) {
        c.jrOptions[method] = append(c.jrOptions[method], o...)
    }
}

type Client struct {
    jrOptions map[string][]jsonrpctransport.ClientOption
    methods   map[string]*jsonrpctransport.Client
}

func NewClient(url string, serviceName string, endpoints []transportlayer.Endpoint, options ...ClientOption) *Client {
    c := &Client{
        jrOptions: make(map[string][]jsonrpctransport.ClientOption),
        methods:   make(map[string]*jsonrpctransport.Client),
    }

    for _, option := range options {
        option(c)
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

        var jrClientOpts []jsonrpctransport.ClientOption
        if jrOpts, ok := c.jrOptions[m.Name()]; ok {
            jrClientOpts = jrOpts
        }
        if jsGlobalOpts, ok := c.jrOptions["*"]; ok {
            jrClientOpts = append(jrClientOpts, jsGlobalOpts...)
        }

        c.methods[m.Name()] = jsonrpctransport.NewClient(
            url,
            serviceName,
            m.Name(),
            converterJSONRPC.EncodeReq,
            converterJSONRPC.DecodeResp,
            converterJSONRPC.ReplyType,
            jrClientOpts...,
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
