package jsonrpc

import (
	"context"
	"net/http"
)

type ServerRequestFunc func(ctx context.Context, r *http.Request) context.Context
