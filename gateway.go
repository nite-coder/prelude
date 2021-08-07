package prelude

import "context"

// Gatewayer handles all communications between client and server
type Gatewayer interface {
	ListenAndServe(bind string, hub Huber) error
	Shutdown(ctx context.Context) error
}

type Item struct {
	Key   string
	Value interface{}
}
