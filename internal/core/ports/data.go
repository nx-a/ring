package ports

import (
	"context"
	"github.com/nx-a/ring/internal/core/domain"
)

type DataService interface {
	Write(ctx context.Context, data domain.Data) error
	Clear()
}
