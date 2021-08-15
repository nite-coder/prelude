package websocket

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	cews "github.com/cloudevents/sdk-go/protocol/ws/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/nite-coder/prelude"
	"github.com/nite-coder/prelude/hub/nats"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateway(t *testing.T) {
	ctx := context.Background()

	opts := nats.HubOptions{
		URL: "nats://nats:4222",
	}
	hub, err := nats.NewNatsHub(opts)
	require.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(3)

	router := prelude.NewRouter("prelude", hub)

	router.AddRoute("hello", func(c *prelude.Context) error {
		defer func() {
			wg.Done()
		}()

		assert.Equal(t, "hello", c.Event.Type())
		_ = c.Set("token", "atoken")
		return c.JSON("wow", "done")
	})

	router.AddRoute("metadata.get", func(c *prelude.Context) error {
		defer func() {
			wg.Done()
		}()

		val := c.Get("token")
		assert.Equal(t, "atoken", val)

		return nil
	})

	go func() {
		websocketGateway := NewGateway()
		err := websocketGateway.ListenAndServe("127.0.0.1:10085", hub)
		require.Nil(t, err)
	}()

	time.Sleep(1 * time.Second)

	// connect to the server
	p, err := cews.Dial(ctx, "ws://127.0.0.1:10085", nil)
	require.NoError(t, err)

	wsClient, err := cloudevents.NewClient(p, cloudevents.WithTimeNow(), cloudevents.WithUUIDs())
	require.NoError(t, err)

	go func() {
		// read message
		received := uint32(0)
		ctx, cancel := context.WithCancel(ctx)
		err := wsClient.StartReceiver(ctx, func(event cloudevents.Event) {
			assert.Equal(t, "wow", event.Type())
			assert.Equal(t, "\"done\"", string(event.Data()))

			sendEvent := cloudevents.NewEvent()
			sendEvent.SetID(uuid.NewString())
			sendEvent.SetSource("client")
			sendEvent.SetType("metadata.get")

			result := wsClient.Send(ctx, sendEvent)
			assert.Equal(t, false, cloudevents.IsUndelivered(result))

			if atomic.AddUint32(&received, 1) == 1 {
				cancel()
			}
		})
		if err != nil {
			log.Printf("failed to start receiver: %v", err)
		} else {
			<-ctx.Done()
		}
		wg.Done()
	}()

	sendEvent := cloudevents.NewEvent()
	sendEvent.SetID(uuid.NewString())
	sendEvent.SetSource("client")
	sendEvent.SetType("hello")
	data := []byte(`{"message":"hello world"}`)
	_ = sendEvent.SetData(cloudevents.ApplicationJSON, data)

	result := wsClient.Send(ctx, sendEvent)
	assert.Equal(t, false, cloudevents.IsUndelivered(result))

	wg.Wait()
}
