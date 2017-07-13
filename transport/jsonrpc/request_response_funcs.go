package jsonrpc

import "context"

type ServerRequestFunc func(ctx context.Context) context.Context
