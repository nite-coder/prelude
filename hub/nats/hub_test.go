package nats

import (
	"sync"
	"testing"

	"github.com/0x5487/prelude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Item struct {
	Message string
}

func TestPublishAndQueueSubscribe(t *testing.T) {
	opts := HubOptions{
		URL: "nats://nats:4222",
	}
	hub, err := NewNatsHub(opts)
	require.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	router := prelude.NewRouter(hub)
	sendCMD := prelude.Command{
		SenderID: "abc123",
		Action:   "sent",
		Data:     []byte(`{"message":"hello world"}`),
	}

	router.AddRoute("s.abc123", func(c *prelude.Context) error {
		assert.Equal(t, "wow", string(c.Command.Action))
		assert.Equal(t, "done", string(c.Command.Data))
		wg.Done()
		return nil
	})

	router.AddRoute("sent", func(c *prelude.Context) error {
		item := Item{}
		err := c.BindJSON(&item)
		require.NoError(t, err)

		assert.Equal(t, "hello world", item.Message)
		assert.Equal(t, sendCMD.SenderID, c.Command.SenderID)
		wg.Done()
		return c.Response("wow", []byte("done"))
	})

	err = hub.Publish(sendCMD.Action, &sendCMD)
	require.Nil(t, err)

	wg.Wait()

}
