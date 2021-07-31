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
		URL:   "nats://nats:4222",
		Group: "gateway",
	}
	hub, err := hubNATS.NewNatsHub(opts)
	if err != nil {
		panic(err)
	}

	router := prelude.NewRouter(hub)
	router.AddRoute("/hello", func(c *prelude.Context) error {
		cmd := c.Command
		log.Str("path", cmd.Path).Str("data", string(cmd.Data)).Debugf("command received")
		return nil
	})

	router.AddRoute("/events/routes_info", func(c *prelude.Context) error {
		cmd := c.Command
		log.Str("path", cmd.Path).Str("data", string(cmd.Data)).Debugf("command session route received")
		return nil
	})

	websocketGateway := websocket.NewGateway()
	err = websocketGateway.ListenAndServe(":10080", hub)
	if err != nil {
		log.Err(err).Error("main: websocket gateway shutdown failed")
	}

}
