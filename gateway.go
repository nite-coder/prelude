package prelude

import "context"

// Gatewayer handles all communications between client and server
type Gatewayer interface {
	ListenAndServe(bind string, hub Huber)
	Shutdown(ctx context.Context) error
}
