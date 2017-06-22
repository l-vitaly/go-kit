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

// MakeServerEndpoints
func MakeServerEndpoints(s interface{}, ef EndpointFactory, of OptionFactory) []Endpoint {
	var endpoints []Endpoint
	structMethods(s, func(method string) {
		e := ef.CreateEndpoint(method)
		options := of.CreateOptions(method)
		if e != nil && options != nil {
			endpoints = append(endpoints, NewEndpoint(method, e, options...))
		}
	})
	return endpoints
}

// MakeClientEndpoints
func MakeClientEndpoints(s interface{}, of OptionFactory) []Endpoint {
	var endpoints []Endpoint
	structMethods(s, func(method string) {
		options := of.CreateOptions(method)
		if options != nil {
			endpoints = append(endpoints, NewEndpoint(method, nil, options...))
		}
	})
	return endpoints
}

func structMethods(s interface{}, fn func(method string)) {
	sType := reflect.TypeOf(s)
	for i := 0; i < sType.NumMethod(); i++ {
		method := sType.Method(i)
		// Method must be exported.
		if method.PkgPath != "" {
			continue
		}
		fn(method.Name)
	}
}
