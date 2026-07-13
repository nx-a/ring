package bucket

import (
	"context"

	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	ctx "github.com/nx-a/ring/internal/engine/context"
	"github.com/nx-a/ring/internal/engine/event"
)

type Service struct {
	stor  ports.BucketStorage
	event *event.Subscriber
}

func New(store ports.BucketStorage, _event *event.Subscriber) *Service {
	return &Service{stor: store, event: _event}
}
func (b *Service) Get(id uint64) (*domain.Bucket, error) {
	backet, err := b.stor.GetById(id)
	if err != nil {
		return nil, err
	}
	return &backet, nil
}

func (b *Service) GetByControl(id uint64) ([]domain.Bucket, error) {
	return b.stor.GetByControlId(id)
}

func (b *Service) Add(bucket *domain.Bucket) *domain.Bucket {
	_b := b.stor.Add(*bucket)
	b.event.Publish("bucket", ctx.WithBucketID(context.Background(), _b.BucketId))
	return &_b
}

func (b *Service) Remove(id uint64) {
	b.stor.Remove(id)
}
func (b *Service) Update(id uint64, bucket domain.Bucket) domain.Bucket {
	return b.stor.Update(id, bucket)
}
func (b *Service) GetAll() []domain.Bucket {
	return b.stor.GetAll()
}
