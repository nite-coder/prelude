package main

import (
	"github.com/0x5487/prelude"
	"github.com/0x5487/prelude/gateway/websocket"
	hubNATS "github.com/0x5487/prelude/hub/nats"
	"github.com/nite-coder/blackbear/pkg/log"
	"github.com/nite-coder/blackbear/pkg/log/handler/console"
)

func main() {
	// use console handler to log all level logs
	logger := log.New()
	clog := console.New()
	logger.AddHandler(clog, log.AllLevels...)
	log.SetLogger(logger)

	// optional: allow handlers to clear all buffer
	defer log.Flush()

	opts := hubNATS.HubOptions{
		URL: "nats://127.0.0.1:4222",
	}
	hub, err := hubNATS.NewNatsHub(opts)
	if err != nil {
		panic(err)
	}

	helloQueue1 := make(chan *prelude.Command, 5)
	err = hub.QueueSubscribe("/hello", "gateway", helloQueue1)

	eventQueue := make(chan *prelude.Command, 5)
	err = hub.QueueSubscribe("/events/routes_info", "gateway", eventQueue)

	go func() {
		for {
			select {
			case cmd := <-helloQueue1:
				log.Str("path", cmd.Path).Str("data", string(cmd.Data)).Debugf("command received")
			case cmd := <-eventQueue:
				log.Str("path", cmd.Path).Str("data", string(cmd.Data)).Debugf("command session route received")
			}
		}
	}()

	websocketGateway := websocket.NewGateway()
	err = websocketGateway.ListenAndServe(":10080", hub)
	if err != nil {
		log.Err(err).Error("main: websocket gateway shutdown failed")
	}

}
