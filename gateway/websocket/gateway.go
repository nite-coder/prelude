package websocket

import (
	"context"

	"github.com/jasonsoft/prelude"
)

// Gateway handles all websocket connections between client and server
type Gateway struct {
}

// NewGateway returns a Gateway instance
func NewGateway() prelude.Gatewayer {
	return &Gateway{}
}

func (g *Gateway) ListenAndServe(bind string, hub prelude.Huber) {
	panic("not implemented") // TODO: Implement
}

func (g *Gateway) Shutdown(ctx context.Context) error {
	panic("not implemented") // TODO: Implement
}
