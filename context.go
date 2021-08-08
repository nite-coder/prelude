package prelude

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

var (
	ErrInvalidEventType = errors.New("prelude: eventType can't be empty")
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
	return c.JSON("metadata.add", item)
}

func (c *Context) BindJSON(obj interface{}) error {
	err := json.Unmarshal(c.Event.Data(), obj)
	if err != nil {
		return err
	}
	return nil
}

func (c *Context) Write(eventType string, bytes []byte) error {
	if eventType == "" {
		return ErrInvalidEventType
	}

	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource(c.hub.Router().name)
	event.SetTime(time.Now().UTC())
	event.SetType(eventType)

	if len(bytes) > 0 {
		err := event.SetData(cloudevents.ApplicationJSON, bytes)
		if err != nil {
			return err
		}
	}

	err := event.Validate()
	if err != nil {
		return err
	}

	sessionID := c.Event.Extensions()[SessionID]
	topic := fmt.Sprintf("s.%s", sessionID)
	return c.hub.Publish(topic, event)
}

func (c *Context) JSON(eventType string, obj interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.Write(eventType, data)
}
