package grapql

import (
	"net/http"

	"golang.org/x/net/context"
)

type RequestFunc func(context.Context, *http.Request) context.Context
