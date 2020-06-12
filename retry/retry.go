package retry

import (
	"time"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/lb"
)

func Endpoint(max int) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return lb.Retry(max, time.Second, lb.NewRoundRobin(sd.FixedEndpointer{next}))
	}
}

func WithCallbackEndpoint(cb lb.Callback) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return lb.RetryWithCallback(time.Second, lb.NewRoundRobin(sd.FixedEndpointer{next}), cb)
	}
}
