package websocket

import (
	"context"
	"encoding/json"
	"hash/fnv"
	"os"
	"sync"
	"sync/atomic"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/nite-coder/blackbear/pkg/config"
	"github.com/nite-coder/blackbear/pkg/log"
	"github.com/nite-coder/prelude"
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
	hub           prelude.Huber
	hostname      string
	ctx           context.Context
	activeState   int32
	mutex         sync.Mutex
	buckets       []*Bucket
	status        *Status
	eventChan     chan cloudevents.Event
	eventStopChan chan bool
}

// NewManager 用來產生一個新的 Manager 用來控制 Gateway
func NewManager(hub prelude.Huber) *Manager {
	hostname, _ := os.Hostname()

	bucketCount, _ := config.Int32("websocket.bucket_count", 128)
	eventCount, _ := config.Int32("websocket.bucket_event_count", 128)

	m := &Manager{
		hub:           hub,
		hostname:      hostname,
		ctx:           context.Background(),
		buckets:       make([]*Bucket, bucketCount),
		status:        &Status{},
		eventChan:     make(chan cloudevents.Event, eventCount),
		eventStopChan: make(chan bool, 1),
		mutex:         sync.Mutex{},
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
	return nil
}

// RouteInfo 代表 session 最後看見的時間
type RouteInfo struct {
	SessionID   string
	GatewayAddr string
	LastSeenAt  time.Time `json:"lastSeenAt"`
}

// UpdateRouteInfo 用來更新目前 session 所在的 gateway 主機和最後一次收到 pong 的時間 (lastSeenAt)
func (m *Manager) UpdateRouteInfo(session *WSSession) error {
	event := cloudevents.NewEvent()
	event.SetID(uuid.NewString())
	event.SetSource(m.hostname)
	event.SetType("events.routes_info")

	body := RouteInfo{
		SessionID:   session.ID(),
		LastSeenAt:  session.LastSeenAt(),
		GatewayAddr: m.hostname,
	}

	b, err := json.Marshal(&body)
	if err != nil {
		return err
	}

	err = event.SetData(cloudevents.ApplicationJSON, b)
	if err != nil {
		return err
	}
	return m.AddEventToHub(event)
}

// Push 用來推播訊息到 client
func (m *Manager) Push(sessionID string, event cloudevents.Event) error {
	if !m.IsActive() {
		log.Debug("websocket: manager can't accept more events because server is shutting down or closed.")
		return nil
	}
	b := m.bucketBySessionID(sessionID)
	return b.push(sessionID, event)
}

// AddEventToHub 把 event 送到 hub 讓 consumer 可以讀取 device 傳送過來的 event
func (m *Manager) AddEventToHub(event cloudevents.Event) error {
	if !m.IsActive() {
		log.Debug("websocket: manager can't accept more events because server is shutting down or closed.")
		return nil
	}

	select {
	case m.eventChan <- event:
	default:
	}

	return nil
}

func (m *Manager) eventLoop() {
	for {
		event := <-m.eventChan
		err := m.hub.Publish(event.Type(), event)
		if err != nil {
			log.Str("action", event.Type()).Error("websocket: fail to publish event to hub")
		}

		if !m.IsActive() && len(m.eventChan) == 0 {
			m.eventStopChan <- true
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

// Start 代表開啟背景工作，例如把 event 送到 hub
func (m *Manager) Start() error {
	m.SetActive(true)
	go m.eventLoop()
	return nil
}

// Shutdown represent graceful shutdown manager
func (m *Manager) Shutdown(ctx context.Context) error {
	m.SetActive(false)
	m.ctx = ctx

	stop := make(chan bool)
	go func() {
		// wait to process all events
		<-m.eventStopChan
		close(m.eventChan)
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
