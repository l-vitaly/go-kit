package nexus

import (
	"context"

	"github.com/gammazero/nexus/client"
	"github.com/gammazero/nexus/wamp"
)

type DecodeRequestFunc func(context.Context, wamp.List) (request interface{}, err error)

// type EncodeRequestFunc func(context.Context, wamp.List) (request interface{}, err error)

type EncodeResponseFunc func(context.Context, interface{}) (response *client.InvokeResult, err error)

// type DecodeResponseFunc func(context.Context, wamp.List) (response interface{}, err error)
