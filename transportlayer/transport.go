package transportlayer

type Endpoints interface {
	Endpoint(m Endpoint)
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

func (t *endpoints) Endpoint(m Endpoint) {
	t.endpoints = append(t.endpoints, m)
}
