package event

import (
	"context"
	log "github.com/sirupsen/logrus"
)

type Subscriber struct {
	listeners map[string][]func(context.Context)
}

func New() *Subscriber {
	return &Subscriber{
		listeners: make(map[string][]func(context.Context)),
	}
}
func (s *Subscriber) On(event string, listener func(context.Context)) {
	if _, ok := s.listeners[event]; !ok {
		s.listeners[event] = make([]func(context.Context), 0, 8)
	}
	s.listeners[event] = append(s.listeners[event], listener)
}
func (s *Subscriber) Publish(event string, ctx context.Context) {
	var listeners []func(context.Context)
	var ok bool
	if listeners, ok = s.listeners[event]; !ok {
		log.Debug("listeners " + event + " not found")
		return
	}
	log.Debug("listeners " + event + " found")
	for _, listener := range listeners {
		go listener(ctx)
	}
}
