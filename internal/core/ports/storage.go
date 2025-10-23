package ports

import (
	"github.com/nx-a/ring/internal/core/domain"
	"time"
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
	Clear(backet string, of time.Time)
	Select(backet string, from time.Time, to time.Time) []domain.Data
	Create(bucket string)
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
	Registration(control domain.Control) (domain.Control, error)
}
