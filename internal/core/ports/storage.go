package ports

import (
	"time"

	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
)

type Storage[T any, I any] interface {
	Add(t T) T
	GetById(id I) (T, error)
	Update(id I, t T) T
	Remove(id I)
}
type PointStorage interface {
	Storage[domain.Point, uint64]
	GetByBacketId(id uint64) []domain.Point
	GetByExternalId(ids []uint64, ext string) *domain.Point
	GetByExternalIds(bucketId uint64, ext []string) []domain.Point
}
type DataStorage interface {
	Add([]domain.Data) error
	Clear(bucketId uint64, of time.Time)
	Select(bucketId uint64, from, to time.Time) []domain.Data
	Create(bucketId uint64)
	Find(data *dto.DataSelect) []domain.Data
	Count(bucketId uint64) (int64, error)
	CountAll() (map[uint64]int64, error)
}
type TokenStorage interface {
	Storage[domain.Token, uint64]
	GetByToken(string) (domain.Token, error)
	GetByBucketId(id uint64) []domain.Token
}
type BucketStorage interface {
	Storage[domain.Bucket, uint64]
	GetByControlId(id uint64) ([]domain.Bucket, error)
	GetAll() []domain.Bucket
}
type ControlStorage interface {
	Auth(login, pass string) (domain.Control, error)
	Registration(control *domain.Control) (domain.Control, error)
}
