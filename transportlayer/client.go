package transportlayer

import "context"

type Client interface {
	Call(ctx context.Context, request interface{}) (response interface{}, err error)
}
