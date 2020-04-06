package prelude

type Command struct {
	SessionID string
	Path      string `json:"path"`
	Type      string
	Data      []byte `json:"data"`
}
