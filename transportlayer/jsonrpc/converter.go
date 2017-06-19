package jsonrpc

import (
	"reflect"

	jsonrpctransport "github.com/l-vitaly/go-kit/transport/jsonrpc"
)

type EndpointServerConverter struct {
	EncodeResp jsonrpctransport.EncodeResponseFunc
	DecodeReq  jsonrpctransport.DecodeRequestFunc
}

type EndpointClientConverter struct {
	EncodeReq  jsonrpctransport.EncodeRequestFunc
	DecodeResp jsonrpctransport.DecodeResponseFunc
	ReplyType  reflect.Type
}
