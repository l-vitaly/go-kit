package rmqrpc

import "golang.org/x/net/context"

// RequestFunc may take information from an RMQ RPC request and put it into a
// request context. In Servers, BeforeFuncs are executed prior to invoking the
// endpoint. In Clients, BeforeFuncs are executed after creating the request
// but prior to invoking the RMQ RPC client.
type RequestFunc func(context.Context) context.Context

// ResponseFunc may take information from a request context.
// ResponseFuncs are only executed in servers, after invoking the endpoint but
// prior to writing a response.
type ResponseFunc func(context.Context)
