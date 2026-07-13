package ports

import (
	"context"

	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
)

type DataService interface {
	Find(ctx context.Context, data *dto.DataSelect) ([]domain.Data, error)
	Write(ctx context.Context, data *domain.Data) error
	Clear()
	Shutdown()
	Count(ctx context.Context, bucketId uint64) (int64, error)
	CountAll(ctx context.Context) (map[uint64]int64, error)
	Subscribe() (<-chan domain.Data, uint64)
	Unsubscribe(id uint64)
}
