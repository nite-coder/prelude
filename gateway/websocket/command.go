package websocket

import (
	"encoding/json"

	"github.com/0x5487/prelude"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

func createCommand(buf []byte) (*prelude.Command, error) {
	var command prelude.Command

	err := json.Unmarshal(buf, &command)
	if err != nil {
		return nil, err
	}

	if command.RequestID == "" {
		command.RequestID = uuid.NewString()
	}

	return &command, nil
}

func toWSMessage(command *prelude.Command) (*WSMessage, error) {
	buf, err := json.Marshal(command)
	if err != nil {
		return nil, err
	}
	msg := &WSMessage{websocket.TextMessage, buf}
	return msg, nil
}

func createEvent(name string, claims map[string]string) *prelude.Command {
	// send addsession event to mq
	cmd := prelude.NewCommand()

	return cmd
}
