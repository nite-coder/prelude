package prelude

type Huber interface {
	Publish(topic string, command *Command) error
	QueueSubscribe(topic string, group string, ch chan *Command) error
	SendCommand(sessionID string, command *Command) error
}
