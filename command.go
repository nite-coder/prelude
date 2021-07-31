package prelude

type Command struct {
	SenderID  string `json:"sender_id"`
	RequestID string `json:"request_id"`
	Path      string `json:"path"`
	Type      string `json:"type"`
	Data      []byte `json:"data"`
}
