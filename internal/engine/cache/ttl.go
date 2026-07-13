package cache

import (
	"sync"
	"time"
)

type item[V any] struct {
	value  V
	expiry time.Time
}

type TTL[K comparable, V any] struct {
	mu      sync.RWMutex
	items   map[K]item[V]
	ttl     time.Duration
	cleanup time.Duration
}

func NewTTL[K comparable, V any](ttl, cleanup time.Duration) *TTL[K, V] {
	c := &TTL[K, V]{
		items:   make(map[K]item[V]),
		ttl:     ttl,
		cleanup: cleanup,
	}
	if cleanup > 0 {
		go c.cleanupLoop()
	}
	return c
}

func (c *TTL[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(it.expiry) {
		var zero V
		return zero, false
	}
	return it.value, true
}

func (c *TTL[K, V]) Set(key K, value V) {
	c.mu.Lock()
	c.items[key] = item[V]{value: value, expiry: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *TTL[K, V]) Delete(key K) {
	c.mu.Lock()
	delete(c.items, key)
	c.mu.Unlock()
}

func (c *TTL[K, V]) cleanupLoop() {
	ticker := time.NewTicker(c.cleanup)
	defer ticker.Stop()
	for range ticker.C {
		c.evictExpired()
	}
}

func (c *TTL[K, V]) evictExpired() {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for k, v := range c.items {
		if now.After(v.expiry) {
			delete(c.items, k)
		}
	}
}
