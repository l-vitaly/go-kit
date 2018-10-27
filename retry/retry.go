package retry

import (
	"context"
	"time"

	"github.com/go-kit/kit/endpoint"
)

// Callback is a function that sets the retry count and an error from the endpoint,
// should return whether Retry will continue to execute the endpoint.
type Callback func(n int, received error) (keepTrying bool)

// Max set max retry.
func Max(max int) Callback {
	return func(n int, err error) (keepTrying bool) {
		return n < max
	}
}

// Always always retry.
func Always() Callback {
	return func(n int, err error) (keepTrying bool) {
		return true
	}
}

// MakeEndpoint create retry middleware.
func MakeEndpoint(delay time.Duration, next endpoint.Endpoint, cb Callback) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (response interface{}, err error) {
		for i := 1; ; i++ {
			response, err = next(ctx, request)
			if err == nil {
				return
			}
			keepTrying := cb(i, err)
			if !keepTrying {
				return
			}
			time.Sleep(delay)
		}
	}
}
