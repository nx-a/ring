package _map

import (
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	"github.com/nx-a/ring/internal/core/domain"
)

func BucketToDomain(bucket *dto.Bucket) *domain.Bucket {
	if bucket == nil {
		return nil
	}
	return &domain.Bucket{
		BucketId:   bucket.BucketId,
		ControlId:  bucket.ControlId,
		SystemName: bucket.SystemName,
		TimeLife:   bucket.TimeLife,
		TimeZone:   bucket.TimeZone,
	}

}
