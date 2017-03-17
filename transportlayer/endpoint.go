package transportlayer

import (
	"context"
	"fmt"
	"time"

	gokitendpoint "github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/metrics"
	"github.com/go-kit/kit/tracing/opentracing"
	"github.com/l-vitaly/eutils"
	opentracinggo "github.com/opentracing/opentracing-go"
)

type Endpoint interface {
	Name() string
	Fn() gokitendpoint.Endpoint
	Converters() []interface{}
}

type EndpointOption func(*endpoint)

type endpoint struct {
	name       string
	fn         gokitendpoint.Endpoint
	converters []interface{}
}

func NewEndpoint(name string, fn gokitendpoint.Endpoint, options ...EndpointOption) Endpoint {
	m := &endpoint{name: name, fn: fn}
	for _, option := range options {
		option(m)
	}
	return m
}

func (m *endpoint) Converters() []interface{} {
	return m.converters
}

func (m *endpoint) Fn() gokitendpoint.Endpoint {
	return m.fn
}

func (m *endpoint) Name() string {
	return m.name
}

func WithConverter(c interface{}) EndpointOption {
	return func(m *endpoint) {
		m.converters = append(m.converters, c)
	}
}

func WithLogger(l log.Logger) EndpointOption {
	return func(m *endpoint) {
		logger := log.With(l, "method", m.name)
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
		histogram := d.With("method", m.name)
		next := m.fn

		m.fn = func(ctx context.Context, request interface{}) (response interface{}, err error) {
			defer func(begin time.Time) {
				histogram.With("success", fmt.Sprint(err == nil)).Observe(time.Since(begin).Seconds())
			}(time.Now())
			return next(ctx, request)
		}
	}
}

func WithTrace(tracer opentracinggo.Tracer) EndpointOption {
	return func(m *endpoint) {
		m.fn = opentracing.TraceServer(tracer, m.name)(m.fn)
	}
}
