package point

import (
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	log "github.com/sirupsen/logrus"
	"sync"
)

type Service struct {
	stor  ports.PointStorage
	bs    ports.BucketService
	cache map[uint64]map[string]*domain.Point
	rw    sync.RWMutex
}

func New(stor ports.PointStorage, bs ports.BucketService) *Service {
	return &Service{stor: stor, bs: bs, cache: make(map[uint64]map[string]*domain.Point)}
}
func (s *Service) Add(controlId uint64, point domain.Point) domain.Point {
	return s.stor.Add(point)
}
func (s *Service) Update(point domain.Point) domain.Point {
	return s.stor.Update(point.PointId, point)
}
func (s *Service) Remove(controlId uint64, pointId uint64) {
	point, err := s.stor.GetById(pointId)
	if err != nil {
		return
	}
	baskets, err := s.bs.GetByControl(controlId)
	if err != nil {
		return
	}
	for _, basket := range baskets {
		if basket.BucketId == point.BucketId {
			s.stor.Remove(pointId)
			break
		}
	}
}
func (s *Service) GetByBacketId(controlId uint64, backetId uint64) []domain.Point {
	baskets, err := s.bs.GetByControl(controlId)
	if err != nil {
		return []domain.Point{}
	}
	find := false
	for _, basket := range baskets {
		if basket.BucketId == backetId {
			find = true
			break
		}
	}
	if !find {
		return []domain.Point{}
	}
	return s.stor.GetByBacketId(backetId)
}
func (s *Service) GetByExternalId(backetId uint64, extId string) domain.Point {
	s.rw.RLock()
	if s.cache[backetId] != nil && s.cache[backetId][extId] != nil {
		defer s.rw.RUnlock()
		return *s.cache[backetId][extId]
	}
	s.rw.RUnlock()
	log.Debug(backetId, extId)
	point := s.stor.GetByExternalId([]uint64{backetId}, extId)
	log.Debug(point)
	if point != nil {
		s.rw.Lock()
		if s.cache[backetId] == nil {
			s.cache[backetId] = make(map[string]*domain.Point)
		}
		s.cache[backetId][extId] = point
		s.rw.Unlock()
		return *point
	}
	return domain.Point{}
}
func (s *Service) GetByExternal(controlId uint64, extId string) domain.Point {
	baskets, err := s.bs.GetByControl(controlId)
	if err != nil {
		return domain.Point{}
	}
	ids := make([]uint64, len(baskets))
	for i, basket := range baskets {
		ids[i] = basket.BucketId
	}
	return *s.stor.GetByExternalId(ids, extId)
}
