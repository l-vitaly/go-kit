package wsjsonrpc

import (
	"encoding/json"

	"context"

	"github.com/go-kit/kit/endpoint"
)

// Server-Side Codec

type EndpointCodecStream struct {
	Endpoint endpoint.Endpoint
	Decode   DecodeStreamRequestFunc
}

type EndpointCodecStreamMap map[string]EndpointCodecStream

type DecodeStreamRequestFunc func(context.Context, json.RawMessage, *Stream) (request interface{}, err error)
