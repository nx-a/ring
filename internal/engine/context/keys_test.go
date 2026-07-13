package ctx

import (
	"context"
	"testing"
)

func TestRequestIDContext(t *testing.T) {
	ctx := context.Background()
	if RequestID(ctx) != "" {
		t.Error("expected empty request id")
	}
	ctx = WithRequestID(ctx, "abc")
	if RequestID(ctx) != "abc" {
		t.Errorf("expected abc, got %q", RequestID(ctx))
	}
}

func TestControlContext(t *testing.T) {
	ctx := context.Background()
	if _, ok := Control(ctx); ok {
		t.Error("expected no control")
	}
	claims := map[string]any{"bucketId": uint64(1)}
	ctx = WithControl(ctx, claims)
	got, ok := Control(ctx)
	if !ok {
		t.Fatal("expected control")
	}
	if got["bucketId"] != uint64(1) {
		t.Errorf("unexpected control: %v", got)
	}
}
