package websocket

import "sync/atomic"

// Status 用來表示 Gateway 的狀態，例如: 連線人數
type Status struct {
	OnlinePeople int64 `json:"online_people"`
}

func (s *Status) increaseOnlinePeople() {
	atomic.AddInt64(&s.OnlinePeople, 1)
}

func (s *Status) decreaseOnlinePeople() {
	atomic.AddInt64(&s.OnlinePeople, -1)
}
