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
	router.AddRoute("ping", func(c *prelude.Context) error {
		return c.Response("pong", []byte("pong"))
	})

	router.AddRoute("hello", func(c *prelude.Context) error {
		content := string(c.Command.Data)
		log.Infof("message from client: %s", content)
		return nil
	})

	websocketGateway := websocket.NewGateway()
	err = websocketGateway.ListenAndServe(":10080", hub)
	if err != nil {
		log.Err(err).Error("main: websocket gateway start failed")
	}
}
