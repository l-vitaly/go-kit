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

type ServerTransportLayer struct {
	endpoints []Endpoint
}

// NewServerTransportLayer
func NewServerTransportLayer(s interface{}, ef EndpointFactory, of OptionFactory) *ServerTransportLayer {
	tl := &ServerTransportLayer{}
	sType := reflect.TypeOf(s)
	for i := 0; i < sType.NumMethod(); i++ {
		method := sType.Method(i)
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		tl.endpoints = append(tl.endpoints, NewEndpoint(method.Name, ef.CreateEndpoint(method.Name), of.CreateOptions(method.Name)...))
	}
	return tl
}

func (t *ServerTransportLayer) GetEndpoints() []Endpoint {
	return t.endpoints
}

type ClientTransportLayer struct {
	endpoints []Endpoint
}

// NewClientTransportLayer
func NewClientTransportLayer(s interface{}, of OptionFactory) *ClientTransportLayer {
	tl := &ClientTransportLayer{}
	sType := reflect.TypeOf(s)
	for i := 0; i < sType.NumMethod(); i++ {
		method := sType.Method(i)
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		tl.endpoints = append(tl.endpoints, NewEndpoint(method.Name, nil, of.CreateOptions(method.Name)...))
	}
	return tl
}

func (t *ClientTransportLayer) GetEndpoints() []Endpoint {
	return t.endpoints
}
