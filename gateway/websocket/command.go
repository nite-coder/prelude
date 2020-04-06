package websocket

import (
	"encoding/json"

	"github.com/gorilla/websocket"
	"github.com/jasonsoft/prelude"
)

// NewCommand 建立一個新 command
func newCommand() *prelude.Command {
	return &prelude.Command{}
}

func createCommand(buf []byte) (*prelude.Command, error) {
	var command prelude.Command

	err := json.Unmarshal(buf, &command)
	if err != nil {
		return nil, err
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
	cmd := newCommand()

	return cmd
}
