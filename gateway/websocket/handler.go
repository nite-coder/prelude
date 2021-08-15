package websocket

import (
	"net/http"

	"github.com/nite-coder/blackbear/pkg/log"
	"github.com/nite-coder/blackbear/pkg/web"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

// RegisterRoute return a router which handles all topics
func RegisterRoute(server *web.WebServer, handler *GatewayHTTPHandler) *web.WebServer {
	server.Get("/", handler.wsEndpoint)
	server.Get("/status", handler.statusEndpoint)
	return server
}

// GatewayHTTPHandler 用來是 Gateway http 的 handler
type GatewayHTTPHandler struct {
	manager *Manager
}

// NewGatewayHTTPHandler 產生一個 GatewayHttpHander instance
func NewGatewayHTTPHandler(manager *Manager) *GatewayHTTPHandler {
	return &GatewayHTTPHandler{
		manager: manager,
	}
}

func (h *GatewayHTTPHandler) wsEndpoint(c *web.Context) error {
	ctx := c.StdContext()
	logger := log.FromContext(ctx)

	defer func() {
		logger.Debug("websocket: wsEndpoint end")
	}()

	respHeader := http.Header{}
	respHeader["Sec-WebSocket-Protocol"] = []string{"cloudevents.json"}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, respHeader)
	if err != nil {
		return err
	}

	sessionID := uuid.NewString()
	clientIP := c.ClientIP()
	wsSession := NewWSSession(sessionID, clientIP, conn, h.manager)
	return wsSession.Start()
}

func (h *GatewayHTTPHandler) statusEndpoint(c *web.Context) error {
	return c.JSON(200, h.manager.status)
}
