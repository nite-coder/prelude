package nats

import (
	"encoding/json"

	"github.com/jasonsoft/prelude"
	natsClient "github.com/nats-io/nats.go"
)

type Hub struct {
	conn *natsClient.Conn
}

type HubOptions struct {
	URL string
}

// TODO: we need to manage connection later in case the connect is disconnected
func NewNatsHub(opts HubOptions) (prelude.Huber, error) {
	nc, err := natsClient.Connect(opts.URL)
	if err != nil {
		return nil, err
	}

	hub := Hub{
		conn: nc,
	}

	return &hub, nil
}

func (hub *Hub) Publish(topic string, command *prelude.Command) error {
	b, err := json.Marshal(command)
	if err != nil {
		return err
	}

	return hub.conn.Publish(topic, b)
}

func (hub *Hub) QueueSubscribe(topic string, group string, ch chan *prelude.Command) error {
	_, err := hub.conn.QueueSubscribe(topic, group, func(msg *natsClient.Msg) {
		cmd := prelude.Command{}
		err := json.Unmarshal(msg.Data, &cmd)
		if err != nil {
			return
		}

		ch <- &cmd
	})

	return err
}

func (hub *Hub) SendCommand(topic string, command *prelude.Command) error {
	return nil
}
