package event

import "github.com/gammazero/nexus/client"

type EventPublisher struct {
	c client.InvocationHandler
}

func NewEventPublisher(c client.InvocationHandler) *EventPublisher {
	return &EventPublisher{}
}
