package cache

import (
	"testing"
	"time"
)

func TestTTLSetGet(t *testing.T) {
	c := NewTTL[string, int](time.Minute, time.Minute)
	c.Set("a", 1)
	if v, ok := c.Get("a"); !ok || v != 1 {
		t.Fatalf("expected 1, got %v, ok=%v", v, ok)
	}
}

func TestTTLExpiration(t *testing.T) {
	c := NewTTL[string, int](10*time.Millisecond, time.Minute)
	c.Set("a", 1)
	if _, ok := c.Get("a"); !ok {
		t.Fatal("expected key to exist")
	}
	time.Sleep(20 * time.Millisecond)
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected key to expire")
	}
}

func TestTTLDelete(t *testing.T) {
	c := NewTTL[string, int](time.Minute, time.Minute)
	c.Set("a", 1)
	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("expected key to be deleted")
	}
}
