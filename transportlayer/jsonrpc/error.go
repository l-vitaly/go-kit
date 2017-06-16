package jsonrpc

import "errors"

var (
	ErrClientEndpointNotFound = errors.New("client endpoint not found")
	ErrServerEndpointNotFound = errors.New("server endpoint not found")
)
