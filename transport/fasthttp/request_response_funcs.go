package fasthttp

import (
	"context"

	"github.com/valyala/fasthttp"
)

// RequestFunc may take information from an HTTP request and put it into a
// request context. In Servers, RequestFuncs are executed prior to invoking the
// endpoint. In Clients, RequestFuncs are executed after creating the request
// but prior to invoking the HTTP client.
type RequestFunc func(context.Context, *fasthttp.Request) context.Context

// RequestFunc may take information from an HTTP request and put it into a
// request context. In Servers, RequestFuncs are executed prior to invoking the
// endpoint. In Clients, RequestFuncs are executed after creating the request
// but prior to invoking the HTTP client.
type ServerRequestFunc func(context.Context, *fasthttp.RequestCtx) context.Context

// ServerResponseFunc may take information from a request context and use it to
// manipulate a ResponseWriter. ServerResponseFuncs are only executed in
// servers, after invoking the endpoint but prior to writing a response.
type ServerResponseFunc func(context.Context, *fasthttp.Response) context.Context

type ClientRequestFunc func(context.Context, *fasthttp.Request) context.Context

// ClientResponseFunc may take information from an HTTP request and make the
// response available for consumption. ClientResponseFuncs are only executed in
// clients, after a request has been made, but prior to it being decoded.
type ClientResponseFunc func(context.Context, *fasthttp.Response) context.Context

// SetContentType returns a ServerResponseFunc that sets the Content-Type header
// to the provided value.
func SetContentType(contentType string) ServerResponseFunc {
	return SetResponseHeader("Content-Type", contentType)
}

// SetResponseHeader returns a ServerResponseFunc that sets the given header.
func SetResponseHeader(key, val string) ServerResponseFunc {
	return func(ctx context.Context, r *fasthttp.Response) context.Context {
		r.Header.Set(key, val)
		return ctx
	}
}

// SetRequestHeader returns a RequestFunc that sets the given header.
func SetRequestHeader(key, val string) ClientRequestFunc {
	return func(ctx context.Context, r *fasthttp.Request) context.Context {
		r.Header.Set(key, val)
		return ctx
	}
}

// PopulateRequestContext is a ServerRequestFunc that populates several values into
// the context from the HTTP request. Those values may be extracted using the
// corresponding ContextKey type in this package.
func PopulateRequestContext(ctx context.Context, rctx *fasthttp.RequestCtx) context.Context {
	for k, v := range map[contextKey][]byte{
		ContextKeyRequestMethod:          rctx.Method(),
		ContextKeyRequestURI:             rctx.RequestURI(),
		ContextKeyRequestPath:            rctx.Path(),
		ContextKeyRequestHost:            rctx.Host(),
		ContextKeyRequestRemoteAddr:      []byte(rctx.RemoteAddr().String()),
		ContextKeyRequestXForwardedFor:   rctx.Request.Header.Peek("X-Forwarded-For"),
		ContextKeyRequestXForwardedProto: rctx.Request.Header.Peek("X-Forwarded-Proto"),
		ContextKeyRequestAuthorization:   rctx.Request.Header.Peek("Authorization"),
		ContextKeyRequestReferer:         rctx.Request.Header.Peek("Referer"),
		ContextKeyRequestUserAgent:       rctx.Request.Header.Peek("User-Agent"),
		ContextKeyRequestXRequestID:      rctx.Request.Header.Peek("X-Request-Id"),
		ContextKeyRequestAccept:          rctx.Request.Header.Peek("Accept"),
	} {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}

type contextKey int

const (
	// ContextKeyRequestMethod is populated in the context by
	// PopulateRequestContext. Its value is r.Method.
	ContextKeyRequestMethod contextKey = iota

	// ContextKeyRequestURI is populated in the context by
	// PopulateRequestContext. Its value is r.RequestURI.
	ContextKeyRequestURI

	// ContextKeyRequestPath is populated in the context by
	// PopulateRequestContext. Its value is r.URL.Path.
	ContextKeyRequestPath

	// ContextKeyRequestHost is populated in the context by
	// PopulateRequestContext. Its value is r.Host.
	ContextKeyRequestHost

	// ContextKeyRequestRemoteAddr is populated in the context by
	// PopulateRequestContext. Its value is r.RemoteAddr.
	ContextKeyRequestRemoteAddr

	// ContextKeyRequestXForwardedFor is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-For").
	ContextKeyRequestXForwardedFor

	// ContextKeyRequestXForwardedProto is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-Proto").
	ContextKeyRequestXForwardedProto

	// ContextKeyRequestAuthorization is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Authorization").
	ContextKeyRequestAuthorization

	// ContextKeyRequestReferer is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Referer").
	ContextKeyRequestReferer

	// ContextKeyRequestUserAgent is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("User-Agent").
	ContextKeyRequestUserAgent

	// ContextKeyRequestXRequestID is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Request-Id").
	ContextKeyRequestXRequestID

	// ContextKeyRequestAccept is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Accept").
	ContextKeyRequestAccept
)
