package transportlayer

import (
    "reflect"

    gokit "github.com/go-kit/kit/endpoint"
)

type EndpointFactory interface {
    CreateEndpoint(m string) gokit.Endpoint
}

type OptionFactory interface {
    CreateOptions(m string) []EndpointOption
}

type TransportLayer struct {
    of        OptionFactory
    ef        EndpointFactory
    endpoints []Endpoint
}

func NewTransportLayer(ef EndpointFactory, cf OptionFactory) *TransportLayer {
    return &TransportLayer{
        of: cf,
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
        t.endpoints = append(t.endpoints, NewEndpoint(method.Name, t.ef.CreateEndpoint(method.Name), t.of.CreateOptions(method.Name)...))
    }
}

func (t *TransportLayer) GetEndpoints() []Endpoint {
    return t.endpoints
}
