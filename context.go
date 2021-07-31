package prelude

import (
	"encoding/json"
	"fmt"
)

type Context struct {
	hub     Huber
	Command *Command
}

func NewContext(hub Huber, cmd *Command) *Context {
	return &Context{
		hub:     hub,
		Command: cmd,
	}
}

func (c *Context) Get(name string) string {
	return ""
}

func (c *Context) Set(key string, val interface{}) {

}

func (c *Context) WriteBytes(name string) string {
	return ""
}

func (c *Context) BindJSON(obj interface{}) error {
	err := json.Unmarshal(c.Command.Data, obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) JSON(obj interface{}) error {
	err := json.Unmarshal(c.Command.Data, obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Response(action string, data []byte) error {
	cmd := Command{
		SenderID:  c.Command.SenderID,
		RequestID: c.Command.RequestID,
		Action:    action,
		Data:      data,
	}

	topic := fmt.Sprintf("/s/%s", c.Command.SenderID)
	return c.hub.Publish(topic, &cmd)
}
