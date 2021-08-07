package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/0x5487/prelude"
	"github.com/0x5487/prelude/hub/nats"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGateway(t *testing.T) {
	opts := nats.HubOptions{
		URL: "nats://nats:4222",
	}
	hub, err := nats.NewNatsHub(opts)
	require.Nil(t, err)

	var wg sync.WaitGroup
	wg.Add(2)

	router := prelude.NewRouter("prelude", hub)
	router.AddRoute("hello", func(c *prelude.Context) error {
		assert.Equal(t, "hello", c.Event.Type())
		_ = c.Set("token", "atoken")
		_ = c.Response("wow", []byte("done"))
		wg.Done()
		return nil
	})

	router.AddRoute("metadata.get", func(c *prelude.Context) error {
		val := c.Get("token")
		assert.Equal(t, "atoken", val)
		wg.Done()
		return nil
	})

	go func() {
		websocketGateway := NewGateway()
		err := websocketGateway.ListenAndServe("127.0.0.1:10085", hub)
		require.Nil(t, err)
	}()

	time.Sleep(1 * time.Second)

	// connect to the server
	ws, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:10085", nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	sendEvent := cloudevents.NewEvent()
	sendEvent.SetID(uuid.NewString())
	sendEvent.SetSource("client")
	sendEvent.SetType("hello")
	data := []byte(`{"message":"hello world"}`)
	_ = sendEvent.SetData(cloudevents.ApplicationJSON, data)

	err = ws.WriteJSON(sendEvent)
	require.Nil(t, err)

	revCMDs := []cloudevents.Event{}
	err = ws.ReadJSON(&revCMDs)
	require.Nil(t, err)

	assert.Equal(t, 1, len(revCMDs))

	revEvent := revCMDs[0]
	assert.Equal(t, "wow", revEvent.Type())
	assert.Equal(t, "done", string(revEvent.Data()))

	sendEvent = cloudevents.NewEvent()
	sendEvent.SetID(uuid.NewString())
	sendEvent.SetSource("client")
	sendEvent.SetType("metadata.get")

	err = ws.WriteJSON(sendEvent)
	require.Nil(t, err)

	wg.Wait()
}
