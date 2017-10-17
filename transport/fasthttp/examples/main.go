package main

import (
	"context"
	"encoding/json"

	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/l-vitaly/go-kit/transport/fasthttp"
	routing "github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

type Service interface {
	Hello(string) string
}

type service struct {
}

func (service) Hello(name string) string {
	return "Hello, " + name
}

func main() {
	router := routing.New()

	svc := &service{}

	server := httptransport.NewServer(
		makeServerEndpoint(svc),
		decodeRequest,
		httptransport.EncodeJSONResponse,
	)

	router.Post("/", server.RouterHandle())

	fasthttp.ListenAndServe(":8080", router.HandleRequest)
}

func makeServerEndpoint(s Service) endpoint.Endpoint {
	return func(_ context.Context, request interface{}) (interface{}, error) {
		req := request.(helloRequest)
		say := s.Hello(req.Name)
		return helloResponse{Say: say}, nil
	}
}

func decodeRequest(_ context.Context, r *fasthttp.Request) (interface{}, error) {
	var req helloRequest
	if err := json.Unmarshal(r.Body(), &req); err != nil {
		return nil, err
	}
	return req, nil
}

type helloRequest struct {
	Name string `json:"name"`
}

type helloResponse struct {
	Say string `json:"say"`
}
