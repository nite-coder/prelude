package websocket

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0x5487/prelude"
	"github.com/nite-coder/blackbear/pkg/config"
	"github.com/nite-coder/blackbear/pkg/log"
)

var fnvHash32 = fnv.New32a()

// FNV32a 用來做切片 string -> int32
func FNV32a(s string) uint32 {
	_, _ = fnvHash32.Write([]byte(s))
	defer fnvHash32.Reset()

	return fnvHash32.Sum32()
}

// Manager 是用來控制 Gateway 的facade
type Manager struct {
	hub             prelude.Huber
	hostname        string
	ctx             context.Context
	activeState     int32
	mutex           sync.Mutex
	buckets         []*Bucket
	status          *Status
	commandChan     chan *prelude.Command
	commandStopChan chan bool
}

// NewManager 用來產生一個新的 Manager 用來控制 Gateway
func NewManager(hub prelude.Huber) *Manager {
	hostname, _ := os.Hostname()

	bucketCount, _ := config.Int32("app.bucket_count", 128)
	commandCount, _ := config.Int32("app.bucket_command_count", 128)

	m := &Manager{
		hub:             hub,
		hostname:        hostname,
		ctx:             context.Background(),
		buckets:         make([]*Bucket, bucketCount),
		status:          &Status{},
		commandChan:     make(chan *prelude.Command, commandCount),
		commandStopChan: make(chan bool, 1),
		mutex:           sync.Mutex{},
	}

	// initial bucket setting
	for idx := range m.buckets {
		m.buckets[idx] = NewBucket(m.ctx, idx, 32)
	}
	return m
}

// Context 獲取 manager 目前的 context
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Status 可以知道目前 Gateway 的狀態，例如連線人數等
func (m *Manager) Status() *Status {
	return m.status
}

// BucketBySessionID 可以找到這個 session 所在的 bucket
func (m *Manager) bucketBySessionID(sessionID string) *Bucket {
	hashNumber := FNV32a(sessionID)
	return m.buckets[hashNumber%uint32(len(m.buckets))]
}

// AddSession 把 session 加到 gateway
func (m *Manager) AddSession(session *WSSession) error {
	bucket := m.bucketBySessionID(session.ID())
	bucket.addSession(session)
	m.status.increaseOnlinePeople()
	return nil
}

// DeleteSession 用來移除 Session
func (m *Manager) DeleteSession(session *WSSession) error {
	bucket := m.bucketBySessionID(session.ID())
	bucket.deleteSession(session)
	m.status.decreaseOnlinePeople()

	cmd := prelude.NewCommand()

	return m.AddCommandToHub(cmd)
}

// RouteInfo 代表 session 最後看見的時間
type RouteInfo struct {
	SessionID   string
	GatewayAddr string
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

// UpdateRouteInfo 用來更新目前 session 所在的 gateway 主機和最後一次收到 pong 的時間 (lastSeenAt)
func (m *Manager) UpdateRouteInfo(session *WSSession) error {
	cmd := prelude.NewCommand()
	cmd.Action = "events.routes_info"

	body := RouteInfo{
		SessionID:   session.ID(),
		LastSeenAt:  session.LastSeenAt(),
		GatewayAddr: m.hostname,
	}

	b, err := json.Marshal(&body)
	if err != nil {
		return err
	}
	cmd.Data = b
	return m.AddCommandToHub(cmd)
}

// Push 用來推播訊息到 client
func (m *Manager) Push(sessionID string, command *prelude.Command) error {
	if !m.IsActive() {
		log.Debug("websocket: manager can't accept more command because manager is shutting down or closed.")
		return nil
	}
	b := m.bucketBySessionID(sessionID)
	return b.push(sessionID, command)
}

// PushAll 廣播訊息到全部 gateway 有連線的 client
func (m *Manager) PushAll(command *prelude.Command) error {
	if !m.IsActive() {
		log.Debug("websocket: manager can't accept more commands because manager is shutting down or closed.")
		return nil
	}
	job := Job{
		OP:      opPushAll,
		Command: command,
	}
	for _, bucket := range m.buckets {
		bucket.jobChan <- job
	}
	return nil
}

// AddCommandToHub 把 command 送到 hub 讓 consumer 可以讀取 device 傳送過來的 command
func (m *Manager) AddCommandToHub(cmd *prelude.Command) error {
	if !m.IsActive() {
		log.Debug("websocket: manager can't accept more commands because manager is shutting down or closed.")
		return nil
	}

	select {
	case m.commandChan <- cmd:
	default:
	}

	return nil
}

func (m *Manager) commandLoop() {
	for {
		cmd := <-m.commandChan
		err := m.hub.Publish(cmd.Action, cmd)
		if err != nil {
			log.Str("path", cmd.Action).Error("websocket: fail to publish command to hub")
		}

		if !m.IsActive() && len(m.commandChan) == 0 {
			m.commandStopChan <- true
		}
	}
}

// IsActive reprsent active status of manager
func (m *Manager) IsActive() bool {
	return atomic.LoadInt32(&(m.activeState)) != 0
}

// SetActive update status of active
func (m *Manager) SetActive(val bool) {
	var i int32 = 0
	if val {
		i = 1
	}
	atomic.StoreInt32(&(m.activeState), int32(i))
}

// Start 代表開啟背景工作，例如把 command 送到 hub
func (m *Manager) Start() error {
	m.SetActive(true)
	go m.commandLoop()
	return nil
}

// Shutdown represent graceful shutdown manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.SetActive(false)
	m.ctx = ctx

	stop := make(chan bool)
	go func() {
		// wait to process all commands
		<-m.commandStopChan
		close(m.commandChan)
		stop <- true
	}()

	select {
	case <-ctx.Done():
		log.Err(ctx.Err()).Error("websocket: manager shutdown timeout")
		break
	case <-stop:
		log.Info("websocket: manager was shutdown gracefully")
	}

	return nil
}
