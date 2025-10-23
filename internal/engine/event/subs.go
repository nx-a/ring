package event

import (
	"context"
	log "github.com/sirupsen/logrus"
	"sync/atomic"
)

type Subscriber struct {
	listeners atomic.Pointer[map[string][]func(context.Context)]
}

func New() *Subscriber {
	s := &Subscriber{}
	emptyMap := make(map[string][]func(context.Context))
	s.listeners.Store(&emptyMap)
	return s
}
func (s *Subscriber) On(event string, listener func(context.Context)) {
	for {
		oldPtr := s.listeners.Load()
		newMap := make(map[string][]func(context.Context), len(*oldPtr)+1)

		// Копируем старые данные
		for k, v := range *oldPtr {
			newMap[k] = v
		}

		// Добавляем новый listener
		if _, ok := newMap[event]; !ok {
			newMap[event] = make([]func(context.Context), 0, 8)
		}
		newMap[event] = append(newMap[event], listener)

		// Пытаемся атомарно обновить
		if s.listeners.CompareAndSwap(oldPtr, &newMap) {
			break
		}
		// Если не получилось, повторяем
	}
}
func (s *Subscriber) Publish(event string, ctx context.Context) {
	listenersMap := s.listeners.Load()
	listeners, ok := (*listenersMap)[event]
	if !ok {
		log.Debug("listeners " + event + " not found")
		return
	}

	log.Debug("listeners " + event + " found")
	for _, listener := range listeners {
		go listener(ctx)
	}
}
