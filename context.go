package prelude

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"time"

	format "github.com/cloudevents/sdk-go/binding/format/protobuf/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/nite-coder/blackbear/pkg/cast"
	"google.golang.org/protobuf/proto"
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

func (c *Context) SenderSessionID() string {
	sessionID, _ := cast.ToString(c.Get(SessionID))
	return sessionID
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

func (c *Context) Write(eventType string, contentType string, bytes []byte, sessionIDs ...string) error {
	if eventType == "" {
		return ErrInvalidEventType
	}

	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource(c.hub.Router().name)
	event.SetTime(time.Now().UTC())
	event.SetType(eventType)

	if len(bytes) > 0 {
		err := event.SetData(contentType, bytes)
		if err != nil {
			return err
		}
	}

	err := event.Validate()
	if err != nil {
		return err
	}

	if len(sessionIDs) == 0 {
		sessionIDs = append(sessionIDs, c.SenderSessionID())
	}

	for _, sessionID := range sessionIDs {
		topic := fmt.Sprintf("sess.%s", sessionID)
		err := c.hub.Publish(topic, event)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Context) JSON(eventType string, obj interface{}, sessionIDs ...string) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	return c.Write(eventType, cloudevents.ApplicationJSON, data)
}

func (c *Context) XML(eventType string, obj interface{}, sessionIDs ...string) error {
	data, err := xml.Marshal(obj)
	if err != nil {
		return err
	}
	return c.Write(eventType, cloudevents.ApplicationXML, data)
}

func (c *Context) ProtoBuf(eventType string, msg proto.Message, sessionIDs ...string) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return err
	}
	return c.Write(eventType, format.ContentTypeProtobuf, data)
}
