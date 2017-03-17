package transportlayer

import "context"

type Server interface {
	Serve(ctx context.Context, req interface{}) (context.Context, interface{}, error)
}
