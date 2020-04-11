package websocket

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jasonsoft/log"
	"github.com/jasonsoft/napnap"
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

// NewRouter return a router which handles all topics
func NewRouter(handler *GatewayHTTPHandler) *napnap.Router {
	router := napnap.NewRouter()
	router.Get("/", handler.wsEndpoint)
	return router
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

func (h *GatewayHTTPHandler) wsEndpoint(c *napnap.Context) {
	ctx := c.StdContext()
	logger := log.FromContext(ctx)

	defer func() {
		logger.Debug("websocket: wsEndpoint end")
	}()

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		panic(err)
	}

	sessionID := uuid.New().String()
	clientIP := c.ClientIP()
	wsSession := NewWSSession(sessionID, clientIP, conn, h.manager)
	_ = wsSession.Start()
}
