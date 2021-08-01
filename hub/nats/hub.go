package nats

import (
	"encoding/json"

	"github.com/0x5487/prelude"
	natsClient "github.com/nats-io/nats.go"
	"github.com/nite-coder/blackbear/pkg/log"
)

type Hub struct {
	router *prelude.Router
	conn   *natsClient.Conn
	group  string
}

type HubOptions struct {
	URL   string
	Group string
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

func (hub *Hub) Router() *prelude.Router {
	return hub.router
}

func (hub *Hub) SetRouter(router *prelude.Router) {
	hub.router = router
}

func (hub *Hub) Publish(topic string, command *prelude.Command) error {
	b, err := json.Marshal(command)
	if err != nil {
		return err
	}

	if command.Action == "" {
		return prelude.ErrInvalidCommand
	}

	err = hub.conn.Publish(topic, b)
	if err != nil {
		log.Err(err).Str("data", string(b)).Error("hub: publish to nats failed")
		return err
	}

	return nil
}

func (hub *Hub) QueueSubscribe(topic string) error {
	_, err := hub.conn.QueueSubscribe(topic, hub.group, func(msg *natsClient.Msg) {
		cmd := prelude.Command{}
		err := json.Unmarshal(msg.Data, &cmd)
		if err != nil {
			return
		}

		h := hub.router.Find(topic)
		c := prelude.NewContext(hub, &cmd)
		_ = h(c)
	})

	return err
}
