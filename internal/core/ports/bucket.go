package ports

import "github.com/nx-a/ring/internal/core/domain"

type BucketService interface {
	GetByControl(id uint64) ([]domain.Bucket, error)
	Get(id uint64) (*domain.Bucket, error)
	Add(bucket *domain.Bucket) *domain.Bucket
	Remove(id uint64)
}
