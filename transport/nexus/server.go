package nexus

import (
	"context"
	"log"

	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/wamp"
	"github.com/go-kit/kit/endpoint"
)

type Server struct {
	ctx context.Context
	e   endpoint.Endpoint
	dec DecodeRequestFunc
	enc EncodeResponseFunc
	// before []RequestFunc
	// after  []ResponseFunc
	logger log.Logger
}

func NewCalle(
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
) client.InvocationHandler {
	return func(ctx context.Context, args wamp.List, kwargs, details wamp.Dict) *client.InvokeResult {
		request, err := dec(ctx, args)
		if err != nil {
			return errorEncoder(err)
		}
		response, err := e(ctx, request)
		if err != nil {
			return errorEncoder(err)
		}
		invokeResult, err := enc(ctx, response)
		if err != nil {
			return errorEncoder(err)
		}
		return invokeResult
	}
}

func errorEncoder(err error) *client.InvokeResult {
	return &client.InvokeResult{
		Err: wamp.URI(err.Error()),
	}
}
