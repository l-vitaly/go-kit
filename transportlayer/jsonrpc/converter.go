package jsonrpc

import (
	jsonrpctransport "github.com/l-vitaly/go-kit/transport/jsonrpc"
)

type EndpointConverter struct {
	EncodeReq  jsonrpctransport.EncodeRequestFunc
	EncodeResp jsonrpctransport.EncodeResponseFunc
	DecodeReq  jsonrpctransport.DecodeRequestFunc
	DecodeResp jsonrpctransport.DecodeResponseFunc
	ReplyType  interface{}
}
