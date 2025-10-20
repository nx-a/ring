package ports

import "github.com/nx-a/ring/internal/core/domain"

type PointService interface {
	GetByExternal(controlId uint64, extId string) domain.Point
	GetByExternalId(backetId uint64, extId string) domain.Point
	GetByBacketId(controlId uint64, backetId uint64) []domain.Point
	Remove(controlId uint64, pointId uint64)
	Update(point domain.Point) domain.Point
	Add(controlId uint64, point domain.Point) domain.Point
}
