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
	"github.com/gorilla/websocket"
	"github.com/nite-coder/blackbear/pkg/config"
	"github.com/nite-coder/blackbear/pkg/log"
)

const (
	// Time allowed to write a message to the peer.
	// writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than readWait.
	pingPeriod = 20 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 2048
)

// WSMessage 代表 websocket 底層的 message
type WSMessage struct {
	MsgType int
	MsgData []byte
}

// WSSession 代表 websocket 每一個 websocket 的連線
type WSSession struct {
	mutex       *sync.Mutex
	activeState int32
	lastSeenAt  time.Time
	manager     *Manager
	clientIP    string

	id          string
	metadata    map[string]interface{}
	socket      *websocket.Conn
	rooms       sync.Map
	roomID      string // member play chatroom and use the roomID
	inChan      chan *WSMessage
	outChan     chan *WSMessage
	commandChan chan *prelude.Command
}

// NewWSSession 產生一個新的 websocket session
func NewWSSession(id string, clientIP string, conn *websocket.Conn, manager *Manager) *WSSession {
	inboundCount, _ := config.Int32("app.session_inbound_count", 128)
	outboundCount, _ := config.Int32("app.session_outbound_count", 128)
	commandCount, _ := config.Int32("app.session_outbound_count", 128)

	return &WSSession{
		mutex:       &sync.Mutex{},
		manager:     manager,
		lastSeenAt:  time.Now().UTC(),
		id:          id,
		socket:      conn,
		inChan:      make(chan *WSMessage, inboundCount),
		outChan:     make(chan *WSMessage, outboundCount),
		commandChan: make(chan *prelude.Command, commandCount),
		clientIP:    clientIP,
		metadata:    make(map[string]interface{}),
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
		close(s.inChan)
		_ = s.Close()
	}()
	s.socket.SetReadLimit(maxMessageSize)
	_ = s.socket.SetReadDeadline(time.Now().Add(pongWait))
	s.socket.SetPongHandler(func(string) error {
		s.lastSeenAt = time.Now().UTC()
		_ = s.socket.SetReadDeadline(time.Now().Add(pongWait)) // Reset the read deadline when a pong is received
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
			log.Debugf("websocket: session id %s readLoop is finished", s.ID())
			return
		}

		msgType, msgData, err = s.socket.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway, websocket.CloseNoStatusReceived, websocket.CloseAbnormalClosure) {
				log.Errorf("websocket: websocket message error: %v", err)
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
		close(s.outChan)
		_ = s.Close()
	}()
	pingTicker := time.NewTicker(pingPeriod)
	var (
		message *WSMessage
		err     error
	)

	for {
		if !s.IsActive() {
			log.Debugf("websocket: session id %s writeLoop is finished", s.ID())
			return
		}
		select {
		case message = <-s.outChan:
			if err = s.socket.WriteMessage(message.MsgType, message.MsgData); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") || errors.Is(err, websocket.ErrCloseSent) {
					log.Err(err).Debug("websocket: wrtieLoop error")
				} else {
					log.Err(err).Error("websocket: wrtieLoop error")
				}
				return
			}
		case <-pingTicker.C:
			if err := s.socket.WriteMessage(websocket.PingMessage, nil); err != nil {
				if strings.Contains(err.Error(), "use of closed network connection") || errors.Is(err, websocket.ErrCloseSent) {
					log.Err(err).Debug("websocket: wrtieLoop ping error")
				} else {
					log.Err(err).Warn("websocket: wrtieLoop ping error")
				}
				return
			}
		}
	}
}

func (s *WSSession) commandLoop() {
	for {
		if !s.IsActive() {
			log.Debugf("websocket: session id %s commandLoop is finished", s.ID())
			return
		}
		select {
		case cmd := <-s.commandChan:
			if cmd != nil {
				// 目前先把 command aggreation 的機制移除，所以不會有 Queue 一秒的問題
				commands := []*prelude.Command{}
				commands = append(commands, cmd)
				buf, err := json.Marshal(commands)
				if err != nil {
					log.Errorf("websocket: command marshal failed: %v", err)
					continue
				}
				message := &WSMessage{websocket.TextMessage, buf}
				s.sendMessage(message)
			}
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

// SendCommand 可以傳送 command 訊息給 client (設備)
func (s *WSSession) SendCommand(cmd *prelude.Command) error {
	_, err := toWSMessage(cmd)
	if err != nil {
		log.Err(err).Error("websocket: command to message fail")
		return err
	}

	select {
	case s.commandChan <- cmd:
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
		log.Debug("websocket: session was closed")
	}
	return nil
}

// Start 代表開始這個 websocket session 開始
func (s *WSSession) Start() error {
	defer func() {
		close(s.commandChan)
		_ = s.Close()
	}()

	s.SetActive(true)

	err := s.manager.AddSession(s)
	if err != nil {
		return err
	}

	go s.readLoop()
	go s.writeLoop()
	go s.commandLoop()
	go s.updateRouteLoop()

	router := s.manager.hub.Router()
	topic := fmt.Sprintf("s.%s", s.ID())
	router.AddRoute(topic, func(c *prelude.Context) error {
		if c.Command.Type == prelude.CommandTypeMetadata {
			item := prelude.Item{}
			err := json.Unmarshal(c.Command.Data, &item)
			if err != nil {
				return err
			}

			switch c.Command.Action {
			case "metadata.add":
				s.metadata[item.Key] = item.Value
				return nil
			}
		}

		return s.SendCommand(c.Command)
	})

	var (
		message    *WSMessage
		commandReq *prelude.Command
	)

	for {
		if !s.IsActive() {
			log.Infof("websocket: session_id: %s start task is finshed", s.ID())
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

		commandReq, err = createCommand(message.MsgData)
		if err != nil {
			log.Err(err).Error("websocket: websocket message is invalid command")
			continue
		}

		commandReq.SenderID = s.ID()
		commandReq.Metadata = s.metadata

		log.Str("action", commandReq.Action).Str("session_id", s.ID()).Str("data", string(commandReq.Data)).Debugf("command sent")

		_ = s.manager.AddCommandToHub(commandReq)
	}
}
