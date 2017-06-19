package transportlayer

import (
	"reflect"

	gokit "github.com/go-kit/kit/endpoint"
)

type EndpointFactory interface {
	CreateEndpoint(m string) gokit.Endpoint
}

type ConverterFactory interface {
	CreateConverters(m string) []interface{}
}

type TransportLayer struct {
	cf        ConverterFactory
	ef        EndpointFactory
	endpoints []Endpoint
}

func (t *TransportLayer) NewTransportLayer(ef EndpointFactory, cf ConverterFactory) *TransportLayer {
	return &TransportLayer{
		cf: cf,
		ef: ef,
	}
}

func (t *TransportLayer) RegisterService(svc interface{}) {
	svcType := reflect.TypeOf(svc)

	for i := 0; i < svcType.NumMethod(); i++ {
		method := svcType.Method(i)

		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}

		t.endpoints = append(t.endpoints, &endpoint{
			name:       method.Name,
			fn:         t.ef.CreateEndpoint(method.Name),
			converters: t.cf.CreateConverters(method.Name),
		})
	}
}

func (t *TransportLayer) GetEndpoints() []Endpoint {
	return t.endpoints
}
