package main

import (
	"github.com/jasonsoft/log"
	"github.com/jasonsoft/log/handlers/console"
	"github.com/jasonsoft/prelude"
	"github.com/jasonsoft/prelude/gateway/websocket"
	hubNATS "github.com/jasonsoft/prelude/hub/nats"
)

func main() {
	// use console handler to log all level logs
	clog := console.New()
	log.RegisterHandler(clog, log.AllLevels...)

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
				fields := log.Fields{
					"path": cmd.Path,
					"data": string(cmd.Data),
				}
				log.WithFields(fields).Debugf("command received")
			case cmd := <-eventQueue:
				fields := log.Fields{
					"path": cmd.Path,
					"data": string(cmd.Data),
				}
				log.WithFields(fields).Debugf("command session route received")
			}
		}
	}()

	websocketGateway := websocket.NewGateway()
	err = websocketGateway.ListenAndServe(":10080", hub)
	if err != nil {
		log.WithError(err).Error("main: websocket gateway shutdown failed")
	}

}
