package transportlayer

type Endpoints interface {
	Endpoint(endpoints ...Endpoint)
	Endpoints() []Endpoint
}

type endpoints struct {
	endpoints []Endpoint
}

func NewEndpoints() Endpoints {
	return &endpoints{}
}

func (t *endpoints) Endpoints() []Endpoint {
	return t.endpoints
}

func (t *endpoints) Endpoint(endpoints ...Endpoint) {
	t.endpoints = endpoints
}
