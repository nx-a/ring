package ctx

import "context"

type key int

const (
	requestIDKey key = iota
	controlKey
	bucketIDKey
)

// RequestID returns the request ID stored in context.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithRequestID adds a request ID to the context.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// Control returns the control claims stored in context.
func Control(ctx context.Context) (map[string]any, bool) {
	c, ok := ctx.Value(controlKey).(map[string]any)
	return c, ok
}

// WithControl adds control claims to the context.
func WithControl(ctx context.Context, control map[string]any) context.Context {
	return context.WithValue(ctx, controlKey, control)
}

// Role returns the role from control claims or empty string.
func Role(ctx context.Context) string {
	c, ok := Control(ctx)
	if !ok {
		return ""
	}
	if r, ok := c["Role"].(string); ok {
		return r
	}
	return ""
}

// IsAdmin reports whether the caller has admin role.
func IsAdmin(ctx context.Context) bool {
	return Role(ctx) == "admin"
}

// WithBucketID adds bucket ID to the context.
func WithBucketID(ctx context.Context, id uint64) context.Context {
	return context.WithValue(ctx, bucketIDKey, id)
}

// BucketID returns the bucket ID stored in context or 0.
func BucketID(ctx context.Context) uint64 {
	if id, ok := ctx.Value(bucketIDKey).(uint64); ok {
		return id
	}
	return 0
}
