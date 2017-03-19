package grapql

import (
	"context"
	"net/http"
)

type DecodeRequestFunc func(ctx context.Context, r *http.Request) (interface{}, error)
