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

func (c *Context) Get(key string) interface{} {
	if c.Command.Metadata == nil {
		return nil
	}
	return c.Command.Metadata[key]
}

func (c *Context) Set(key string, val interface{}) error {
	item := Item{
		Key:   key,
		Value: val,
	}

	data, err := json.Marshal(item)
	if err != nil {
		return err
	}

	cmd := Command{
		SenderID:  c.Command.SenderID,
		RequestID: c.Command.RequestID,
		Action:    "metadata.add",
		Type:      CommandTypeMetadata,
		Data:      data,
	}

	topic := fmt.Sprintf("s.%s", c.Command.SenderID)
	return c.hub.Publish(topic, &cmd)
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

	topic := fmt.Sprintf("s.%s", c.Command.SenderID)
	return c.hub.Publish(topic, &cmd)
}
