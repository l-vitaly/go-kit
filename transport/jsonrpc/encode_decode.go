package jsonrpc

import (
	"context"
)

type DecodeRequestFunc func(context.Context, interface{}) (request interface{}, err error)

type EncodeRequestFunc func(context.Context, interface{}) (request interface{}, err error)

type EncodeResponseFunc func(context.Context, interface{}) (response interface{}, err error)

type DecodeResponseFunc func(context.Context, interface{}) (response interface{}, err error)
