package test

import (
	"context"
	"net"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/l-vitaly/eutils"
	"github.com/l-vitaly/go-kit/test/pb"
	"github.com/l-vitaly/go-kit/transportlayer"
	transportgrpc "github.com/l-vitaly/go-kit/transportlayer/grpc"
	"github.com/l-vitaly/gounit"
	context2 "golang.org/x/net/context"
	"google.golang.org/grpc"
)

type MethodNameRequest struct {
	Param1 string
}

type MethodNameResponse struct {
	Result string
	Err    error
}

type NameService interface {
	MethodName(ctx context.Context, param1 string) (string, error)
}

type service struct {
}

func (*service) MethodName(ctx context.Context, param1 string) (string, error) {
	return "hello", nil
}

type server struct {
	ts transportlayer.Server
}

func (s *server) MethodName(ctx context2.Context, req *pb.MethodNameRequest) (*pb.MethodNameResponse, error) {
	_, resp, err := s.ts.Serve(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp.(*pb.MethodNameResponse), nil
}

type client struct {
	tc transportlayer.Client
}

func (c *client) MethodName(ctx context.Context, param1 string) (string, error) {
	resp, err := c.tc.Call(ctx, &MethodNameRequest{Param1: param1})
	if err != nil {
		return "", err
	}
	return resp.(*MethodNameResponse).Result, resp.(*MethodNameResponse).Err
}

func TestTransportLayer(t *testing.T) {
	u := gounit.New(t)

	logger := log.NewJSONLogger(os.Stderr)

	svc := &service{}
	endpoints := []transportlayer.Endpoint{
		transportlayer.NewEndpoint(
			"MethodName",
			func(ctx context.Context, request interface{}) (interface{}, error) {
				req := request.(*MethodNameRequest)
				res, err := svc.MethodName(ctx, req.Param1)
				return &MethodNameResponse{Result: res, Err: err}, nil
			},
			transportlayer.WithLogger(logger),
			transportlayer.WithConverter(
				&transportgrpc.EndpointConverter{
					func(_ context.Context, request interface{}) (interface{}, error) {
						req := request.(*MethodNameRequest)
						return &pb.MethodNameRequest{Param1: req.Param1}, nil
					},
					func(_ context.Context, response interface{}) (interface{}, error) {
						resp := response.(*MethodNameResponse)
						return &pb.MethodNameResponse{Result: resp.Result, Err: eutils.Err2Str(resp.Err)}, nil
					},
					func(_ context.Context, request interface{}) (interface{}, error) {
						req := request.(*pb.MethodNameRequest)
						return &MethodNameRequest{Param1: req.Param1}, nil
					},
					func(_ context.Context, response interface{}) (interface{}, error) {
						resp := response.(*pb.MethodNameResponse)
						return &MethodNameResponse{Result: resp.Result, Err: eutils.Str2Err(resp.Err)}, nil
					},
					pb.MethodNameResponse{},
				},
			),
		),
	}

	go func() {
		listener, err := net.Listen("tcp", ":50505")
		if err != nil {
			panic(err)
			return
		}

		grpcs := grpc.NewServer()
		pb.RegisterNameServer(grpcs, &server{transportgrpc.NewServer(endpoints...)})

		grpcs.Serve(listener)
	}()

	conn, _ := grpc.Dial(":50505", grpc.WithInsecure())
	defer conn.Close()

	c := &client{transportgrpc.NewClient("Name", conn, endpoints...)}

	res, err := c.MethodName(context.Background(), "")
	u.AssertNotError(err, "Call MethodName")
	u.AssertEquals("hello", res, "Result MethodName")
}
