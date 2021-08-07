package prelude

import (
	"encoding/json"
	"fmt"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

type Context struct {
	hub   Huber
	Event cloudevents.Event
}

func NewContext(hub Huber, event cloudevents.Event) *Context {
	return &Context{
		hub:   hub,
		Event: event,
	}
}

func (c *Context) Get(key string) interface{} {
	return c.Event.Extensions()[key]
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

	return c.Response("metadata.add", data)
}

func (c *Context) WriteBytes(name string) string {
	return ""
}

func (c *Context) BindJSON(obj interface{}) error {
	err := json.Unmarshal(c.Event.Data(), obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) JSON(obj interface{}) error {
	err := json.Unmarshal(c.Event.Data(), obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Response(action string, data []byte) error {
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource(c.hub.Router().name)
	event.SetType(action)
	err := event.SetData(cloudevents.ApplicationJSON, data)
	if err != nil {
		return err
	}

	err = event.Validate()
	if err != nil {
		return err
	}

	sessionID := c.Event.Extensions()["sessionid"]
	topic := fmt.Sprintf("s.%s", sessionID)
	return c.hub.Publish(topic, event)
}
