package prelude

import cloudevents "github.com/cloudevents/sdk-go/v2"

type Huber interface {
	Router() *Router
	SetRouter(router *Router)
	Publish(topic string, event cloudevents.Event) error
	QueueSubscribe(topic string) error
}
