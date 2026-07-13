package logger

import (
	"context"

	"github.com/google/uuid"
	ctx "github.com/nx-a/ring/internal/engine/context"
	log "github.com/sirupsen/logrus"
)

// NewRequestID generates a new unique request ID.
func NewRequestID() string {
	return uuid.New().String()
}

// FromContext returns a logrus entry with request_id field.
func FromContext(c context.Context) *log.Entry {
	return log.WithField("request_id", ctx.RequestID(c))
}
