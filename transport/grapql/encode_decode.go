package grapql

import (
	"net/http"

	"golang.org/x/net/context"
)

type DecodeRequestFunc func(ctx context.Context, r *http.Request) (interface{}, error)
