# prelude
A simple lightweight network framework for Go.  It is useful for long connection applications such as `IOT`, `Chatroom`, `Online Game`, `Instant Messaging` 

## Feature
1. distributed architecture and can be scale out 
1. handles 1m connections
1. `Websocket` is supported (`TCP`, `MQTT` maybe later)
1. use the `CloudEvents 1.0 specification` as message format
1. Golang style

## Roadmap
1. support stand-alone architecture via channel
1. different encoding format, such as `PROTOBUF`, `JSON`
1. support middleware chain


## Example

#### Server
```Go
func main() {
	opts := hubNATS.HubOptions{
		URL:   "nats://nats:4222",
		Group: "gateway",
	}

    // we use nats as mq
	hub, err := hubNATS.NewNatsHub(opts)
	if err != nil {
		panic(err)
	}

	router := prelude.NewRouter(hub)
	router.AddRoute("ping", func(c *prelude.Context) error {
		return c.JSON("pong", "hello world") // the event will send back to sender
	})

	websocketGateway := websocket.NewGateway()
	err = websocketGateway.ListenAndServe(":10080", hub)
	if err != nil {
		log.Err(err).Error("main: websocket gateway start failed")
	}
}

```

#### Client
```Go

func main() {
	ws, _, err := websocket.DefaultDialer.Dial("ws://127.0.0.1:10080", nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer ws.Close()

	pingEvent := cloudevents.NewEvent()
	pingEvent.SetID(uuid.NewString())
	pingEvent.SetSource("client")
	pingEvent.SetType("ping")

	ws.WriteJSON(sendEvent)
}

```
