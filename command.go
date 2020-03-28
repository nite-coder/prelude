package prelude

type Command struct {
	SessionID string
	Type      string
	Timestamp int64
	Data      []byte
}
