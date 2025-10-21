package _map

import (
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	"github.com/nx-a/ring/internal/core/domain"
)

func PointToDomain(point *dto.Point) *domain.Point {
	if point == nil {
		return nil
	}
	return &domain.Point{
		PointId:    point.PointId,
		BucketId:   point.BucketId,
		ExternalId: point.ExternalId,
		TimeZone:   point.TimeZone,
	}
}
