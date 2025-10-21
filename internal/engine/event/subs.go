package event

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Subscriber struct {
	listeners map[string][]func(context.Context)
	rwMux     sync.RWMutex
}

func New() *Subscriber {
	return &Subscriber{
		listeners: make(map[string][]func(context.Context)),
	}
}
func (s *Subscriber) On(event string, listener func(context.Context)) {
	s.rwMux.Lock()
	defer s.rwMux.Unlock()
	if _, ok := s.listeners[event]; !ok {
		s.listeners[event] = make([]func(context.Context), 0, 8)
	}
	s.listeners[event] = append(s.listeners[event], listener)
}
func (s *Subscriber) Publish(event string, ctx context.Context) {
	s.rwMux.RLocker()
	var listeners []func(context.Context)
	var ok bool
	if listeners, ok = s.listeners[event]; !ok {
		s.rwMux.RUnlock()
		log.Debug("listeners " + event + " not found")
		return
	}
	s.rwMux.RUnlock()
	log.Debug("listeners " + event + " found")
	for _, listener := range listeners {
		go listener(ctx)
	}
}
