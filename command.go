package prelude

import "encoding/json"

type Command struct {
	SessionID string          `json:"session_id"`
	Path      string          `json:"path"`
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data"`
}
