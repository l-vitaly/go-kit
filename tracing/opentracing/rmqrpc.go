package opentracing

import (
	"github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
)

func FromRMQRPCRequest(tracer opentracing.Tracer, operationName string) func(ctx context.Context) context.Context {
	return func(ctx context.Context) context.Context {
		span := tracer.StartSpan(operationName)
		return opentracing.ContextWithSpan(ctx, span)
	}
}
