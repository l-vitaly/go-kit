package jsonrpc

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"reflect"

	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/l-vitaly/jsonrpc/json2"
)

type ClientOption func(*Client)

func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers[key] = value
	}
}

func WithProxy(proxy func(*http.Request) (*url.URL, error)) ClientOption {
	return func(c *Client) {
		c.transport.Proxy = proxy
	}
}

func WithDialContext(dial func(ctx context.Context, network, addr string) (net.Conn, error)) ClientOption {
	return func(c *Client) {
		c.transport.DialContext = dial
	}
}

func WithDialTLS(dial func(network, addr string) (net.Conn, error)) ClientOption {
	return func(c *Client) {
		c.transport.DialTLS = dial
	}
}

func WithTLSClientConfig(tlsConfig *tls.Config) ClientOption {
	return func(c *Client) { c.transport.TLSClientConfig = tlsConfig }
}

func WithTLSHandshakeTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.transport.TLSHandshakeTimeout = d
	}
}

func WithKeepAlives(keepAlives bool) ClientOption {
	return func(c *Client) {
		c.transport.DisableKeepAlives = !keepAlives
	}
}

func WithCompression(compress bool) ClientOption {
	return func(c *Client) {
		c.transport.DisableCompression = !compress
	}
}

func WithMaxIdleConns(maxIdleConns int) ClientOption {
	return func(c *Client) {
		c.transport.MaxIdleConns = maxIdleConns
	}
}

func WithMaxIdleConnsPerHost(maxIdleConnsPerHost int) ClientOption {
	return func(c *Client) {
		c.transport.MaxIdleConnsPerHost = maxIdleConnsPerHost
	}
}

func WithIdleConnTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.transport.IdleConnTimeout = d
	}
}

func WithResponseHeaderTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.transport.ResponseHeaderTimeout = d
	}
}

func WithExpectContinueTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.transport.ExpectContinueTimeout = d
	}
}

func WithTLSNextProto(tlsNextProto map[string]func(authority string, c *tls.Conn) http.RoundTripper) ClientOption {
	return func(c *Client) {
		c.transport.TLSNextProto = tlsNextProto
	}
}

func WithProxyConnectHeader(proxyConnectHeader http.Header) ClientOption {
	return func(c *Client) {
		c.transport.ProxyConnectHeader = proxyConnectHeader
	}
}

func WithMaxResponseHeaderBytes(maxResponseHeaderBytes int64) ClientOption {
	return func(c *Client) {
		c.transport.MaxResponseHeaderBytes = maxResponseHeaderBytes
	}
}

type Client struct {
	url          string
	serviceName  string
	method       string
	enc          EncodeRequestFunc
	dec          DecodeResponseFunc
	token        string
	transport    *http.Transport
	headers      map[string]string
	jsonRPCReply interface{}
}

func NewClient(
	url string,
	serviceName string,
	method string,
	enc EncodeRequestFunc,
	dec DecodeResponseFunc,
	jsonRPCReply interface{},
	options ...ClientOption,
) *Client {
	c := &Client{
		url:          url,
		serviceName:  serviceName,
		method:       method,
		enc:          enc,
		dec:          dec,
		transport:    &http.Transport{},
		headers:      make(map[string]string),
		jsonRPCReply: jsonRPCReply,
	}
	for _, o := range options {
		o(c)
	}
	return c
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

		httpClient := &http.Client{Transport: c.transport}

		httpReq, err := http.NewRequest("POST", c.url, bytes.NewReader(reqBytes))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		for key, value := range c.headers {
			httpReq.Header.Set(key, value)
		}
		respRpc, err := httpClient.Do(httpReq)
		if err != nil {
			return nil, err
		}
		if respRpc.Body != nil {
			defer respRpc.Body.Close()
		}
		if respRpc.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Resonse status code by %d", respRpc.StatusCode)
		}

		jsonRPCReply := reflect.New(
			reflect.TypeOf(c.jsonRPCReply),
		).Interface()

		err = json2.DecodeClientResponse(respRpc.Body, jsonRPCReply)
		if err != nil {
			return nil, err
		}

		jsonRPCReply = reflect.Indirect(reflect.ValueOf(jsonRPCReply)).Interface()

		response, err := c.dec(ctx, jsonRPCReply)
		if err != nil {
			return nil, fmt.Errorf("Decode: %v", err)
		}
		return response, nil
	}
}
