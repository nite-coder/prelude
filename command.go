package prelude

type CommandType int32

const (
	CommandTypeMetadata = 1
)

type Command struct {
	SenderID  string                 `json:"sender_id"`
	Source    string                 `json:"source"`
	RequestID string                 `json:"request_id"`
	Action    string                 `json:"action"`
	Type      CommandType            `json:"type"`
	Data      []byte                 `json:"data"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func NewCommand() *Command {
	return &Command{
		Metadata: make(map[string]interface{}),
	}
}

type Item struct {
	Key   string
	Value interface{}
}
