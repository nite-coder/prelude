package websocket

import (
	"sync"
	"testing"
	"time"

	"github.com/0x5487/prelude"
	"github.com/0x5487/prelude/hub/nats"
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
	wg.Add(1)

	router := prelude.NewRouter(hub)
	router.AddRoute("/hello", func(c *prelude.Context) error {
		assert.Equal(t, "/hello", string(c.Command.Path))
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

	// Send message to server, read response and check to see if it's what we expect.
	sendCMD := prelude.Command{
		Path: "/hello",
		Data: []byte(`{"message":"hello world"}`),
	}

	err = ws.WriteJSON(sendCMD)
	require.Nil(t, err)

	wg.Wait()

}
