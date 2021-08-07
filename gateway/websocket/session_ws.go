package websocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0x5487/prelude"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/gorilla/websocket"
	"github.com/nite-coder/blackbear/pkg/config"
	"github.com/nite-coder/blackbear/pkg/log"
)

const (
	// Time allowed to write a message to the peer.
	// writeWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than readWait.
	pingPeriod = 20 * time.Second
)

// WSMessage 代表 websocket 底層的 message
type WSMessage struct {
	MsgType int
	MsgData []byte
}

// WSSession 代表 websocket 每一個 websocket 的連線
type WSSession struct {
	mutex       sync.Mutex
	activeState int32
	lastSeenAt  time.Time
	manager     *Manager
	clientIP    string

	id        string
	metadata  map[string]interface{}
	socket    *websocket.Conn
	rooms     sync.Map
	roomID    string // member play chatroom and use the roomID
	inChan    chan *WSMessage
	outChan   chan *WSMessage
	eventChan chan cloudevents.Event
}

// NewWSSession 產生一個新的 websocket session
func NewWSSession(id string, clientIP string, conn *websocket.Conn, manager *Manager) *WSSession {
	inboundCount, _ := config.Int32("app.session_inbound_count", 128)
	outboundCount, _ := config.Int32("app.session_outbound_count", 128)
	eventCount, _ := config.Int32("app.session_event_count", 128)

	return &WSSession{
		manager:    manager,
		lastSeenAt: time.Now().UTC(),
		id:         id,
		socket:     conn,
		inChan:     make(chan *WSMessage, inboundCount),
		outChan:    make(chan *WSMessage, outboundCount),
		eventChan:  make(chan cloudevents.Event, eventCount),
		clientIP:   clientIP,
		metadata:   make(map[string]interface{}),
	}
}

// ID 可以取得 session 的唯一值
func (s *WSSession) ID() string {
	return s.id
}

// LastSeenAt 取得 session 的最後獲得 pong 的時間
func (s *WSSession) LastSeenAt() time.Time {
	return s.lastSeenAt
}

// Metadata returns session's metadata
func (s *WSSession) Metadata() map[string]interface{} {
	return s.metadata
}

// IsActive reprsent active status of manager
func (s *WSSession) IsActive() bool {
	return atomic.LoadInt32(&(s.activeState)) != 0
}

// SetActive update status of active
func (s *WSSession) SetActive(val bool) {
	var i int32 = 0
	if val {
		i = 1
	}
	atomic.StoreInt32(&(s.activeState), int32(i))
}

func (s *WSSession) readLoop() {
	defer func() {
		_ = s.Close()
	}()

	pongWaitSec, _ := config.Duration("app.pong_wait_sec", 0)
	pongWait := pongWaitSec * time.Second
	if pongWaitSec > 0 {
		_ = s.socket.SetReadDeadline(time.Now().Add(pongWait))
	}

	// Maximum message size allowed from peer.
	maxMessageSizeByte, _ := config.Int64("app.max_message_size_byte", 0)
	if maxMessageSizeByte > 0 {
		s.socket.SetReadLimit(maxMessageSizeByte)
	}

	s.socket.SetPongHandler(func(string) error {
		s.lastSeenAt = time.Now().UTC()
		if pongWaitSec > 0 {
			_ = s.socket.SetReadDeadline(time.Now().Add(pongWait)) // Reset the read deadline when a pong is received
		}
		return nil
	})

	var (
		msgType int
		msgData []byte
		message *WSMessage
		err     error
	)

	for {
		if !s.IsActive() {
			log.Str("session_id", s.ID()).Debugf("websocket: session id %s readLoop is finished", s.ID())
			return
		}

		msgType, msgData, err = s.socket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
				log.Err(err).Errorf("websocket: read websocket message failed")
			}
			return
		}

		message = &WSMessage{
			MsgType: msgType,
			MsgData: msgData,
		}

		s.inChan <- message
	}
}

func (s *WSSession) writeLoop() {
	defer func() {
		_ = s.Close()
	}()
	pingTicker := time.NewTicker(pingPeriod)
	var (
		message *WSMessage
		err     error
	)

	for {
		if !s.IsActive() {
			log.Str("session_id", s.ID()).Debugf("websocket: session id %s writeLoop is finished", s.ID())
			return
		}
		select {
		case message = <-s.outChan:
			if err = s.socket.WriteMessage(message.MsgType, message.MsgData); err != nil {
				if !(strings.Contains(err.Error(), "use of closed network connection") || errors.Is(err, websocket.ErrCloseSent)) {
					log.Err(err).Warn("websocket: wrtieLoop error")
				} else {
					log.Err(err).Debug("websocket: wrtieLoop error")
				}
				return
			}
		case <-pingTicker.C:
			if err := s.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				if !(strings.Contains(err.Error(), "use of closed network connection") || errors.Is(err, websocket.ErrCloseSent)) {
					log.Err(err).Warn("websocket: wrtieLoop ping error")
				} else {
					log.Err(err).Debug("websocket: wrtieLoop ping error")
				}
				return
			}
		}
	}
}

func (s *WSSession) eventLoop() {
	for {
		if !s.IsActive() {
			log.Str("session_id", s.ID()).Debugf("websocket: session id %s eventLoop is finished", s.ID())
			return
		}
		select {
		case event := <-s.eventChan:
			// 目前先把 event aggreation 的機制移除，所以不會有 Queue 一秒的問題
			events := []cloudevents.Event{}
			events = append(events, event)
			buf, err := json.Marshal(events)
			if err != nil {
				log.Errorf("websocket: event marshal failed: %v", err)
				continue
			}
			message := &WSMessage{websocket.TextMessage, buf}
			s.sendMessage(message)
		}
		// case <-timer:
		// 	//log.Debugf("command chan length: %d", len(s.commandChan))
		// 	if len(commands) > 0 {
		// 		buf, err := json.Marshal(commands)
		// 		commands = []*gateway.Command{}
		// 		if err != nil {
		// 			log.Errorf("websocket: command marshal failed: %v", err)
		// 			continue
		// 		}
		// 		message := &WSMessage{websocket.TextMessage, buf}
		// 		s.sendMessage(message)
		// 	}
		// }
	}
}

func (s *WSSession) updateRouteLoop() {
	timer := time.NewTicker(time.Duration(60) * time.Second)
	log.Debug("websocket: updateRouteLoop is started")

	for range timer.C {
		if !s.IsActive() {
			log.Debugf("websocket: updateRouteLoop is finished")
			return
		}

		_ = s.manager.UpdateRouteInfo(s)

		log.Str("session_id", s.ID()).Str("last_seen_at", s.lastSeenAt.String()).Debug("websocket: session route was updated")
	}
}

func (s *WSSession) readMessage() *WSMessage {
	message := <-s.inChan
	return message
}

func (s *WSSession) sendMessage(msg *WSMessage) {
	select {
	case s.outChan <- msg:
	default:
	}
}

// SendEvent 可以傳送 event 訊息給 client (設備)
func (s *WSSession) SendEvent(event cloudevents.Event) error {
	_, err := toWSMessage(event)
	if err != nil {
		log.Err(err).Error("websocket: event to webscoket message fail")
		return err
	}

	select {
	case s.eventChan <- event:
	default:
	}

	return nil
}

// Close func which closes websocket session and remove session from bucket and room.
func (s *WSSession) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.IsActive() {
		_ = s.socket.Close()
		_ = s.manager.DeleteSession(s)
		s.SetActive(false)
		log.Str("session_id", s.ID()).Debug("websocket: session was closed")
	}

	return nil
}

// Start 代表開始這個 websocket session 開始
func (s *WSSession) Start() error {
	defer func() {
		close(s.inChan)
		close(s.outChan)
		close(s.eventChan)
		_ = s.Close()
	}()

	s.SetActive(true)

	err := s.manager.AddSession(s)
	if err != nil {
		return err
	}

	go s.readLoop()
	go s.writeLoop()
	go s.eventLoop()

	isUpdateRoute, _ := config.Bool("app.session_update_route", false)
	if isUpdateRoute {
		go s.updateRouteLoop()
	}

	router := s.manager.hub.Router()
	topic := fmt.Sprintf("s.%s", s.ID())
	router.AddRoute(topic, func(c *prelude.Context) error {

		switch c.Event.Type() {
		case "metadata.add":
			item := prelude.Item{}
			err := json.Unmarshal(c.Event.Data(), &item)
			if err != nil {
				return err
			}
			s.metadata[item.Key] = item.Value
			return nil
		}

		return s.SendEvent(c.Event)
	})

	var (
		message *WSMessage
	)

	for {
		if !s.IsActive() {
			log.Str("session_id", s.ID()).Infof("websocket: session_id: %s start task is finshed", s.ID())
			return nil
		}

		message = s.readMessage()
		if message == nil {
			continue // the channel might be closed
		}

		if message.MsgType != websocket.TextMessage {
			log.Info("websocket: message type is not text message")
			continue
		}

		event := cloudevents.NewEvent()
		err = json.Unmarshal(message.MsgData, &event)
		if err != nil {
			log.Err(err).Str("data", string(message.MsgData)).Warn("websocket: websocket message is invalid.")
			continue
		}

		err = event.Validate()
		if err != nil {
			log.Err(err).Str("data", string(message.MsgData)).Warn("websocket: event is invalid from client")
			continue
		}

		event.SetExtension("sessionid", s.ID())
		for k, v := range s.metadata {
			event.SetExtension(k, v)
		}

		log.Str("action", event.Type()).Str("session_id", s.ID()).Str("data", string(event.Data())).Debugf("event was received from client")
		_ = s.manager.AddEventToHub(event)
	}
}

func toWSMessage(event cloudevents.Event) (*WSMessage, error) {
	buf, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	msg := &WSMessage{websocket.TextMessage, buf}
	return msg, nil
}
