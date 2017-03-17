package transportlayer

import (
	"context"
	"fmt"
	"time"

	gokitendpoint "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	grpctransport "github.com/go-kit/kit/transport/grpc"
	"github.com/l-vitaly/eutils"
)

type endpointConverter struct {
	req  interface{}
	resp interface{}
}

func (mc endpointConverter) Request() interface{} {
	return mc.req
}

func (mc endpointConverter) Response() interface{} {
	return mc.resp
}

type Endpoint interface {
	Name() string
	Fn() gokitendpoint.Endpoint
	Reply() interface{}
	Encode() endpointConverter
	Decode() endpointConverter
}

type EndpointOption func(*endpoint)

type endpoint struct {
	name   string
	fn     gokitendpoint.Endpoint
	decode endpointConverter
	encode endpointConverter
	reply  interface{}
}

func NewEndpoint(name string, fn gokitendpoint.Endpoint, reply interface{}, options ...EndpointOption) Endpoint {
	m := &endpoint{name: name, fn: fn, reply: reply}
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *endpoint) Reply() interface{} {
	return m.reply
}

func (m *endpoint) Decode() endpointConverter {
	return m.decode
}

func (m *endpoint) Encode() endpointConverter {
	return m.encode
}

func (m *endpoint) Fn() gokitendpoint.Endpoint {
	return m.fn
}

func (m *endpoint) Name() string {
	return m.name
}

func WithEncode(req grpctransport.EncodeRequestFunc, resp grpctransport.EncodeResponseFunc) EndpointOption {
	return func(m *endpoint) {
		m.encode = endpointConverter{req: req, resp: resp}
	}
}

func WithDecode(req grpctransport.DecodeRequestFunc, resp grpctransport.DecodeResponseFunc) EndpointOption {
	return func(m *endpoint) {
		m.decode = endpointConverter{req: req, resp: resp}
	}
}

func WithLogger(l log.Logger) EndpointOption {
	return func(m *endpoint) {
		logger := log.With(l, "endpoint", m.name)
		next := m.fn

		m.fn = func(ctx context.Context, request interface{}) (resp interface{}, err error) {
			defer func(begin time.Time) {
				logger.Log("error", eutils.Err2Str(err), "took", time.Since(begin))
			}(time.Now())
			return next(ctx, request)
		}
	}
}

func WithDuration(d metrics.Histogram) EndpointOption {
	return func(m *endpoint) {
		histogram := d.With("endpoint", m.name)
		next := m.fn

		m.fn = func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				histogram.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}
