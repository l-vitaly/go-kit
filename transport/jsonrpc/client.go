package jsonrpc

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"reflect"

	"github.com/go-kit/kit/endpoint"
	"github.com/l-vitaly/jsonrpc/json2"
)

type Client struct {
	url          string
	serviceName  string
	method       string
	enc          EncodeRequestFunc
	dec          DecodeResponseFunc
	jsonRPCReply reflect.Type
}

func NewClient(
	url string,
	serviceName string,
	method string,
	enc EncodeRequestFunc,
	dec DecodeResponseFunc,
	jsonRPCReply reflect.Type,
) *Client {
	return &Client{
		url:          url,
		serviceName:  serviceName,
		method:       method,
		enc:          enc,
		dec:          dec,
		jsonRPCReply: jsonRPCReply,
	}
}

func (c Client) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		req, err := c.enc(ctx, request)
		if err != nil {
			return nil, fmt.Errorf("Encode: %v", err)
		}

		reqBytes, err := json2.EncodeClientRequest(fmt.Sprintf("%s.%s", c.serviceName, c.method), req)
		if err != nil {
			return nil, err
		}

		respRpc, err := http.Post(c.url, "application/json", bytes.NewReader(reqBytes))
		if err != nil {
			return nil, err
		}

		jsonRPCReply := reflect.New(c.jsonRPCReply).Interface()

		err = json2.DecodeClientResponse(respRpc.Body, jsonRPCReply)
		if err != nil {
			return nil, err
		}

		response, err := c.dec(ctx, jsonRPCReply)
		if err != nil {
			return nil, fmt.Errorf("Decode: %v", err)
		}
		return response, nil
	}
}
