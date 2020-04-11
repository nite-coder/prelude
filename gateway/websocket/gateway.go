package websocket

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasonsoft/log"
	"github.com/jasonsoft/napnap"
	"github.com/jasonsoft/prelude"
)

// Gateway handles all websocket connections between client and server
type Gateway struct {
	manager *Manager
}

// NewGateway returns a Gateway instance
func NewGateway() prelude.Gatewayer {
	return &Gateway{}
}

func (g *Gateway) ListenAndServe(bind string, hub prelude.Huber) error {
	g.manager = NewManager(hub)

	nap := napnap.New()
	corsOpts := napnap.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedHeaders: []string{"*"},
	}
	nap.Use(napnap.NewCors(corsOpts))
	nap.Use(napnap.NewHealth())

	router := napnap.NewRouter()
	router.Get("/ping", func(c *napnap.Context) {
		_ = c.String(http.StatusOK, "["+c.ClientIP()+"] pong!!!")
	})
	nap.Use(router)

	gatewayHTTPhandler := NewGatewayHTTPHandler(g.manager)
	gatewayRouter := NewRouter(gatewayHTTPhandler)
	nap.Use(gatewayRouter)

	httpServer := &http.Server{
		Addr:    bind,
		Handler: nap,
	}

	go func() {
		// service connections
		log.Infof("websocket: Listening and serving HTTP on %s\n", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if err == http.ErrServerClosed {
			log.Infof("websocket: http server closed under request: %v", err)
		} else {
			log.Fatalf("websocket: http server closed unexpect: %v", err)
		}
	}()

	go func() {
		_ = g.manager.Start()
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGKILL, syscall.SIGHUP, syscall.SIGTERM)
	<-stopChan
	log.Info("websocket: shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := g.manager.Shutdown(ctx); err != nil {
		log.Errorf("websocket: gateway manager shutdown error: %v", err)
	} else {
		log.Info("websocket: gateway manager gracefully stopped")
	}

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Errorf("websocket: http server shutdown error: %v", err)
	} else {
		log.Info("websocket: http gracefully stopped")
	}

	return nil
}

func (g *Gateway) Shutdown(ctx context.Context) error {
	panic("not implemented") // TODO: Implement
}
