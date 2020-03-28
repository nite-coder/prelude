package nats

import (
	"testing"

	"github.com/jasonsoft/prelude"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishAndQueueSubscribe(t *testing.T) {
	queue := make(chan *prelude.Command, 10)

	opts := HubOptions{
		URL: "nats://127.0.0.1:4222",
	}
	hub, err := NewNatsHub(opts)
	require.Nil(t, err)

	err = hub.QueueSubscribe("session_id", "gateway", queue)
	require.Nil(t, err)

	cmd := prelude.Command{
		SessionID: "abbc",
		Type:      "cmd",
		Data:      []byte("Hello World"),
	}

	err = hub.Publish("session_id", &cmd)
	assert.Nil(t, err)

	revCmd := <-queue

	assert.Equal(t, "abbc", revCmd.SessionID)
	assert.Equal(t, "cmd", revCmd.Type)
	assert.Equal(t, "cmd", revCmd.Type)
	assert.Equal(t, "Hello World", string(revCmd.Data))
}
