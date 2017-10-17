package rmqrpc

import "context"

// DecodeRequestFunc extracts a user-domain request object from a RMQ RPC request.
// It's designed to be used in RMQ RPC servers, for server-side endpoints. One
// straightforward DecodeRequestFunc could be something that
// decodes from the RMQ RPC request message to the concrete request type.
type DecodeRequestFunc func(context.Context, interface{}) (request interface{}, err error)

// EncodeRequestFunc encodes the passed request object into the RMQ RPC request
// object. It's designed to be used in RMQ RPC clients, for client-side
// endpoints. One straightforward EncodeRequestFunc could something that
// encodes the object directly to the RMQ RPC request message.
type EncodeRequestFunc func(context.Context, interface{}) (request interface{}, err error)

// EncodeResponseFunc encodes the passed response object to the RMQ RPC response
// message. It's designed to be used in RMQ RPC servers, for server-side
// endpoints. One straightforward EncodeResponseFunc could be something that
// encodes the object directly to the RMQ RPC response message.
type EncodeResponseFunc func(context.Context, interface{}) (response interface{}, err error)

// DecodeResponseFunc extracts a user-domain response object from a RMQ RPC
// response object. It's designed to be used in RMQ RPC clients, for client-side
// endpoints. One straightforward DecodeResponseFunc could be something that
// decodes from the RMQ RPC response message to the concrete response type.
type DecodeResponseFunc func(context.Context, interface{}) (response interface{}, err error)
