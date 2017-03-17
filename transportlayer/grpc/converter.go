package grpc

import (
	grpctransport "github.com/go-kit/kit/transport/grpc"
)

type EndpointConverter struct {
	EncodeReq  grpctransport.EncodeRequestFunc
	EncodeResp grpctransport.EncodeResponseFunc
	DecodeReq  grpctransport.DecodeRequestFunc
	DecodeResp grpctransport.DecodeResponseFunc
	ReplyType  interface{}
}
