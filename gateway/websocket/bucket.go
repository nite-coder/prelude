package websocket

import (
	"context"
	"sync"

	"github.com/jasonsoft/log"
	"github.com/jasonsoft/prelude"
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
		jobChan: make(chan Job, 1000),
	}
	// create workers
	for i := 0; i < workerCount; i++ {
		go b.doJob(ctx)
	}
	return b
}

func (b *Bucket) addSession(session *WSSession) {
	b.sessions.Store(session.ID(), session)
	log.Infof("service: session id %s was added to bucket id %d", session.ID(), b.id)
}

func (b *Bucket) deleteSession(session *WSSession) {
	b.sessions.Delete(session.ID())
	log.Infof("service: session id %s was deleted from bucket id %d", session.ID(), b.id)
}

func (b *Bucket) pushAll(command *prelude.Command) {
	var (
		session *WSSession
		ok      bool
	)

	b.sessions.Range(func(key, value interface{}) bool {
		session, ok = value.(*WSSession)
		if ok {
			msg, err := toWSMessage(command)
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

func (b *Bucket) push(sessionID string, command *prelude.Command) error {
	session := b.session(sessionID)
	if session == nil {
		// session was close or not exist
		log.Debugf("service: session_id: %s doesn't exist", sessionID)
		return nil
	}
	log.Debugf("service: session_id: %s was found", sessionID)
	return session.SendCommand(command)
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
				_ = b.push(job.SessionID, job.Command)
			case opPushAll:
				b.pushAll(job.Command)
			}
		}
	}
}
