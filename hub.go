package prelude

type Huber interface {
	Router() *Router
	SetRouter(router *Router)
	Publish(topic string, command *Command) error
	QueueSubscribe(topic string) error
	SendCommand(sessionID string, command *Command) error
}
