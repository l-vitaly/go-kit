package retry

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
)

func Endpoint(max int, timeout time.Duration) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return lb.Retry(max, timeout, lb.NewRoundRobin(sd.FixedEndpointer{next}))
	}
}

func WithCallbackEndpoint(cb lb.Callback, timeout time.Duration) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return lb.RetryWithCallback(timeout, lb.NewRoundRobin(sd.FixedEndpointer{next}), cb)
	}
}
