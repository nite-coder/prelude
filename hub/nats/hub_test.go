package nats

import (
	"sync"
	"testing"

	"github.com/0x5487/prelude"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
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

	router := prelude.NewRouter("prelude", hub)

	sessionID := "my_session_id"

	router.AddRoute("s."+sessionID, func(c *prelude.Context) error {
		defer func() {
			wg.Done()
		}()

		assert.Equal(t, "wow", string(c.Event.Type()))
		assert.Equal(t, "\"done\"", string(c.Event.Data()))

		return nil
	})

	router.AddRoute("sent", func(c *prelude.Context) error {
		defer func() {
			wg.Done()
		}()

		item := Item{}
		err := c.BindJSON(&item)
		require.NoError(t, err)

		assert.Equal(t, "hello world", item.Message)
		assert.Equal(t, sessionID, c.Event.Extensions()[prelude.SessionID])

		return c.JSON("wow", "done")
	})

	sendEvent := cloudevents.NewEvent()
	sendEvent.SetID(uuid.NewString())
	sendEvent.SetSource("client")
	sendEvent.SetType("sent")
	data := []byte(`{"message":"hello world"}`)
	_ = sendEvent.SetData(cloudevents.ApplicationJSON, data)
	sendEvent.SetExtension(prelude.SessionID, sessionID)

	err = hub.Publish(sendEvent.Type(), sendEvent)
	require.Nil(t, err)

	wg.Wait()

}
