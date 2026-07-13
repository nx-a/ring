package ports

import "context"

type StatusService interface {
	Status(ctx context.Context) (map[string]any, error)
	Metrics(ctx context.Context) (map[string]any, error)
}
