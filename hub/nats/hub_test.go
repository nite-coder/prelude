package nats

import (
	"testing"

	"github.com/jasonsoft/prelude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishAndQueueSubscribe(t *testing.T) {
	opts := HubOptions{
		URL: "nats://127.0.0.1:4222",
	}
	hub, err := NewNatsHub(opts)
	require.Nil(t, err)

	queue1 := make(chan *prelude.Command)
	err = hub.QueueSubscribe("s/session_1", "gateway", queue1)
	require.Nil(t, err)

	queue2 := make(chan *prelude.Command, 1)
	err = hub.QueueSubscribe("s/session_2", "gateway", queue2)
	require.Nil(t, err)

	cmd := prelude.Command{
		SessionID: "abbc",
		Type:      "cmd",
		Data:      []byte("Hello World"),
	}

	err = hub.Publish("s/session_1", &cmd)
	require.Nil(t, err)

	revCmd := <-queue1

	assert.Equal(t, "abbc", revCmd.SessionID)
	assert.Equal(t, "cmd", revCmd.Type)
	assert.Equal(t, "Hello World", string(revCmd.Data))

}
