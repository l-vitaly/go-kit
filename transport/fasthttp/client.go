package fasthttp

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/url"
	"strings"
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/valyala/fasthttp"
)

type FastHTTPClient interface {
	Do(req *fasthttp.Request, resp *fasthttp.Response) error
}

// Client wraps a URL and provides a method that implements endpoint.Endpoint.
type Client struct {
	client  *fasthttp.Client
	method  string
	tgt     *url.URL
	timeout time.Duration
	enc     EncodeRequestFunc
	dec     DecodeResponseFunc
	before  []ClientRequestFunc
	after   []ClientResponseFunc
}

// NewClient constructs a usable Client for a single remote method.
func NewClient(
	method string,
	tgt *url.URL,
	enc EncodeRequestFunc,
	dec DecodeResponseFunc,
	options ...ClientOption,
) *Client {
	c := &Client{
		client: &fasthttp.Client{},
		method: method,
		tgt:    tgt,
		enc:    enc,
		dec:    dec,
		before: []ClientRequestFunc{},
		after:  []ClientResponseFunc{},
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// ClientOption sets an optional parameter for clients.
type ClientOption func(*Client)

// SetClient sets the underlying HTTP client used for requests.
// By default, http.DefaultClient is used.
func SetClient(client *fasthttp.Client) ClientOption {
	return func(c *Client) { c.client = client }
}

// SetClientTimeout sets the client request timeout .
func SetClientTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) { c.timeout = timeout }
}

// ClientBefore sets the RequestFuncs that are applied to the outgoing HTTP
// request before it's invoked.
func ClientBefore(before ...ClientRequestFunc) ClientOption {
	return func(c *Client) { c.before = append(c.before, before...) }
}

// ClientAfter sets the ClientResponseFuncs applied to the incoming HTTP
// request prior to it being decoded. This is useful for obtaining anything off
// of the response and adding onto the context prior to decoding.
func ClientAfter(after ...ClientResponseFunc) ClientOption {
	return func(c *Client) { c.after = append(c.after, after...) }
}

// Endpoint returns a usable endpoint that invokes the remote endpoint.
func (c Client) Endpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		var (
			err error
		)

		req := fasthttp.AcquireRequest()
		req.SetRequestURI(c.tgt.String())
		req.Header.SetMethod(strings.ToUpper(c.method))
		defer fasthttp.ReleaseRequest(req)

		resp := fasthttp.AcquireResponse()
		defer fasthttp.ReleaseResponse(resp)

		if err = c.enc(ctx, req, request); err != nil {
			return nil, err
		}

		for _, f := range c.before {
			ctx = f(ctx, req)
		}

		err = c.client.Do(req, resp)

		if err != nil {
			return nil, err
		}

		for _, f := range c.after {
			ctx = f(ctx, resp)
		}

		response, err := c.dec(ctx, resp)
		if err != nil {
			return nil, err
		}

		return response, nil
	}
}

// EncodeJSONRequest is an EncodeRequestFunc that serializes the request as a
// JSON object to the Request body. Many JSON-over-HTTP services can use it as
// a sensible default. If the request implements Headerer, the provided headers
// will be applied to the request.
func EncodeJSONRequest(c context.Context, r *fasthttp.Request, request interface{}) error {
	r.Header.Set("Content-Type", "application/json; charset=utf-8")
	if headerer, ok := request.(Headerer); ok {
		for k, v := range headerer.Headers() {
			r.Header.Set(k, v)
		}
	}
	b, err := json.Marshal(request)
	if err != nil {
		return err
	}
	r.SetBody(b)
	return nil
}

// EncodeXMLRequest is an EncodeRequestFunc that serializes the request as a
// XML object to the Request body. If the request implements Headerer,
// the provided headers will be applied to the request.
func EncodeXMLRequest(c context.Context, r *fasthttp.Request, request interface{}) error {
	r.Header.Set("Content-Type", "text/xml; charset=utf-8")
	if headerer, ok := request.(Headerer); ok {
		for k, v := range headerer.Headers() {
			r.Header.Set(k, v)
		}
	}
	b, err := xml.Marshal(request)
	if err != nil {
		return err
	}
	r.SetBody(b)
	return nil
}
