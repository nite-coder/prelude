package websocket

import (
	"context"
	"sync"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/nite-coder/blackbear/pkg/log"
)

// Bucket 是水桶，用來加速查詢用
type Bucket struct {
	id       int
	rooms    sync.Map
	sessions sync.Map
	jobChan  chan Job
}

// NewBucket 會生成一個新的 Bucket
func NewBucket(ctx context.Context, id, workerCount int) *Bucket {
	b := &Bucket{
		id:      id,
		jobChan: make(chan Job, 128),
	}
	// create workers
	for i := 0; i < workerCount; i++ {
		go b.doJob(ctx)
	}
	return b
}

func (b *Bucket) addSession(session *WSSession) {
	b.sessions.Store(session.ID(), session)
	log.Str("session_id", session.ID()).Infof("service: session id %s was added to bucket id %d", session.ID(), b.id)
}

func (b *Bucket) deleteSession(session *WSSession) {
	b.sessions.Delete(session.ID())
	log.Str("session_id", session.ID()).Infof("service: session id %s was deleted from bucket id %d", session.ID(), b.id)
}

func (b *Bucket) pushAll(event cloudevents.Event) {
	var (
		session *WSSession
		ok      bool
	)

	b.sessions.Range(func(key, value interface{}) bool {
		session, ok = value.(*WSSession)
		if ok {
			msg, err := toWSMessage(event)
			if err != nil {
				return true
			}
			session.sendMessage(msg)
		}
		return true
	})
}

func (b *Bucket) session(sessionID string) *WSSession {
	session, found := b.sessions.Load(sessionID)
	if found {
		session, ok := session.(*WSSession)
		if ok {
			return session
		}
	}
	return nil
}

func (b *Bucket) push(sessionID string, event cloudevents.Event) error {
	session := b.session(sessionID)
	if session == nil {
		// session was close or not exist
		log.Str("session_id", sessionID).Debugf("service: session_id: %s doesn't exist", sessionID)
		return nil
	}
	log.Str("session_id", sessionID).Debugf("service: session_id: %s was found", sessionID)
	return session.SendEvent(event)
}

func (b *Bucket) count() int {
	length := 0
	b.sessions.Range(func(_, _ interface{}) bool {
		length++
		return true
	})
	return length
}

// doJob function which will be executed by workers.
func (b *Bucket) doJob(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			log.Errorf("service: doJob is timeout: %v", ctx.Err())
			break
		case job := <-b.jobChan:
			switch job.OP {
			case opPush:
				_ = b.push(job.SessionID, job.Event)
			case opPushAll:
				b.pushAll(job.Event)
			}
		}
	}
}
