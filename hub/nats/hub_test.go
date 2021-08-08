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
	sessionID := "my_session_id"

	opts := HubOptions{
		URL: "nats://nats:4222",
	}
	hub, err := NewNatsHub(opts)
	require.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	router := prelude.NewRouter("prelude", hub)

	router.AddRoute("s."+sessionID, func(c *prelude.Context) error {
		defer func() {
			wg.Done()
		}()

		assert.Equal(t, "pong", string(c.Event.Type()))
		assert.Equal(t, "\"done\"", string(c.Event.Data()))

		return nil
	})

	router.AddRoute("ping", func(c *prelude.Context) error {
		defer func() {
			wg.Done()
		}()

		item := Item{}
		err := c.BindJSON(&item)
		require.NoError(t, err)

		assert.Equal(t, "hello world", item.Message)
		assert.Equal(t, cloudevents.ApplicationJSON, c.Event.DataContentType())
		assert.Equal(t, sessionID, c.Get(prelude.SessionID))

		return c.JSON("pong", "done")
	})

	pingEvent := cloudevents.NewEvent()
	pingEvent.SetID(uuid.NewString())
	pingEvent.SetSource("client")
	pingEvent.SetType("ping")
	pingEvent.SetExtension(prelude.SessionID, sessionID)

	data := []byte(`{"message":"hello world"}`)
	err = pingEvent.SetData(cloudevents.ApplicationJSON, data)
	require.NoError(t, err)

	err = hub.Publish(pingEvent.Type(), pingEvent)
	require.NoError(t, err)

	wg.Wait()
}
