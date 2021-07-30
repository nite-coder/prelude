package websocket

import "github.com/0x5487/prelude"

const (
	opPush    = 1
	opPushAll = 2 // push to all online members
)

// Job 用來表示 gateway 需要做的任務，例如傳送訊息給 client
type Job struct {
	OP        int
	SessionID string
	Command   *prelude.Command
	WSMessage *WSMessage
}
