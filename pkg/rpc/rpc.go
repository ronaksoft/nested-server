package rpc

import (
	"git.ronaksoftware.com/ronak/toolbox"
	"sync"
)

/*
   Creation Time: 2019 - Mar - 13
   Created by:  (ehsan)
   Maintainers:
      1.  Ehsan N. Moosa (E2)
   Auditor: Ehsan N. Moosa (E2)
   Copyright Ronak Software Group 2018
*/

// SimpleWorker
type SimpleWorker struct {
	sync.Mutex
	rateLimiter         ronak.RateLimiter
	handlers            map[MessageConstructor]MessageHandler
	handlersWithSession map[MessageConstructor]MessageHandlerWithSession
}

func NewSimpleRPCWorker(rateLimiter ronak.RateLimiter) *SimpleWorker {
	w := new(SimpleWorker)
	w.rateLimiter = rateLimiter
	w.handlers = make(map[MessageConstructor]MessageHandler)
	w.handlersWithSession = make(map[MessageConstructor]MessageHandlerWithSession)
	return w
}

func (w *SimpleWorker) AddHandler(constructor MessageConstructor, handler MessageHandler) {
	w.Lock()
	defer w.Unlock()
	w.handlers[constructor] = handler
}

func (w *SimpleWorker) AddHandlerWithSession(constructor MessageConstructor, handler MessageHandlerWithSession) {
	w.Lock()
	defer w.Unlock()
	w.handlersWithSession[constructor] = handler
}

func (w *SimpleWorker) SetHandlers(handlers map[MessageConstructor]MessageHandler) {
	w.Lock()
	defer w.Unlock()

	w.handlers = handlers
}

func (w *SimpleWorker) SetHandlersWithSession(handlers map[MessageConstructor]MessageHandlerWithSession) {
	w.Lock()
	defer w.Unlock()

	w.handlersWithSession = handlers
}

func (w *SimpleWorker) Execute(msg Message) (Message, error) {
	if err := w.rateLimiter.Enter(); err != nil {
		return msg, err
	}
	defer w.rateLimiter.Leave()

	if h, ok := w.handlers[msg.Constructor]; !ok {
		return msg, ErrNoHandler
	} else {
		return h(msg), nil
	}
}

func (w *SimpleWorker) ExecuteWithSession(sessionID int64, msg Message) (Message, error) {
	if err := w.rateLimiter.Enter(); err != nil {
		return msg, err
	}
	defer w.rateLimiter.Leave()

	if sh, ok := w.handlersWithSession[msg.Constructor]; !ok {
		if h, ok := w.handlers[msg.Constructor]; !ok {
			return msg, ErrNoHandler
		} else {
			return h(msg), nil
		}

	} else {
		return sh(sessionID, msg), nil
	}
}

// messageHandler
// It accepts a ronak.SimpleSession object and Message as inputs and returns Message as output
type MessageHandler func(in Message) (out Message)
type MessageHandlerWithSession func(sessionID int64, in Message) (out Message)
type AdvancedMessageHandler func(userID, sessionID int64, in Message) (out Message)
