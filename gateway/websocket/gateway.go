package websocket

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nite-coder/blackbear/pkg/log"
	"github.com/nite-coder/blackbear/pkg/web"
	"github.com/nite-coder/blackbear/pkg/web/middleware"
	"github.com/nite-coder/prelude"
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
	// Increase resources limitations
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return err
	}
	rLimit.Cur = rLimit.Max
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return err
	}

	g.manager = NewManager(hub)

	s := web.NewServer()
	corsOpts := middleware.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
		AllowedHeaders: []string{"*"},
	}
	s.Use(middleware.NewCors(corsOpts))
	s.Use(middleware.NewHealth())

	s.Get("/ping", func(c *web.Context) error {
		_ = c.String(http.StatusOK, "["+c.ClientIP()+"] pong!!!")
		return nil
	})

	gatewayHTTPhandler := NewGatewayHTTPHandler(g.manager)
	s = RegisterRoute(s, gatewayHTTPhandler)

	go func() {
		// service connections
		log.Infof("websocket: Listening and serving HTTP on %s\n", bind)
		err := s.Run(bind)
		if errors.Is(err, http.ErrServerClosed) {
			log.Infof("websocket: http server closed under request: %v", err)
		} else {
			log.Fatalf("websocket: http server closed unexpect: %v", err)
		}
	}()

	go func() {
		_ = g.manager.Start()
	}()

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	<-stopChan
	log.Info("websocket: shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := g.manager.Shutdown(ctx); err != nil {
		log.Errorf("websocket: gateway manager shutdown error: %v", err)
	} else {
		log.Info("websocket: gateway manager gracefully stopped")
	}

	if err := s.Shutdown(ctx); err != nil {
		log.Errorf("websocket: web server shutdown error: %v", err)
	} else {
		log.Info("websocket: web server gracefully stopped")
	}
	return nil
}

func (g *Gateway) Shutdown(ctx context.Context) error {
	return g.manager.Shutdown(ctx)
}
