package rmq

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"
	"github.com/l-vitaly/rmqrpc"
	pb "github.com/l-vitaly/rmqrpc/proto"
)

// Client wraps a RMQ RPC connection and provides a method that implements
// endpoint.Endpoint.
type Client struct {
	client      rmqrpc.Client
	serviceName string
	method      string
	enc         EncodeRequestFunc
	dec         DecodeResponseFunc
	rmqReply    interface{}
	before      []RequestFunc
}

// NewClient constructs a usable Client for a single remote endpoint.
// Pass an zero-value struct message of the RMQ RPC response type as
// the rmqReply argument.
func NewClient(
	c rmqrpc.Client,
	serviceName string,
	method string,
	enc EncodeRequestFunc,
	dec DecodeResponseFunc,
	rmqReply interface{},
	options ...ClientOption,
) *Client {
	return &Client{
		client:      c,
		serviceName: serviceName,
		method:      method,
		enc:         enc,
		dec:         dec,
		rmqReply:    rmqReply,
	}
}

// ClientOption sets an optional parameter for clients.
type ClientOption func(*Client)

// ClientBefore sets the RequestFuncs that are applied to the outgoing RMQ RPC
// request before it's invoked.
func ClientBefore(before ...RequestFunc) ClientOption {
	return func(c *Client) { c.before = before }
}

// Endpoint returns a usable endpoint that will invoke the RMQ RPC specified by the
// client.
func (c Client) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		req, err := c.enc(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("Encode: %v", err)
		}

		for _, f := range c.before {
			ctx = f(ctx)
		}

		out, err := c.client.Invoke(ctx, fmt.Sprint("%s.%s", c.serviceName, c.method), req, true, c.rmqReply)
		if err != nil {
			return nil, fmt.Errorf("Exec: %v", err)
		}

		rmqReply := <-out

		replyErr, ok := rmqReply.(*pb.Error)

		if ok {
			return nil, fmt.Errorf("Reply error: %v Code: %d", replyErr, replyErr.Code)
		}

		response, err := c.dec(ctx, rmqReply)
		if err != nil {
			return nil, fmt.Errorf("Decode: %v", err)
		}
		return response, nil
	}
}
