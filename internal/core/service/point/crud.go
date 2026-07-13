package point

import (
	"sync"
	"time"

	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/engine/cache"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	stor  ports.PointStorage
	bs    ports.BucketService
	cache *cache.TTL[uint64, map[string]*domain.Point]
	rw    sync.RWMutex
}

func New(stor ports.PointStorage, bs ports.BucketService) *Service {
	return &Service{
		stor:  stor,
		bs:    bs,
		cache: cache.NewTTL[uint64, map[string]*domain.Point](5*time.Minute, 10*time.Minute),
	}
}

func (s *Service) Add(controlId uint64, point domain.Point) domain.Point {
	p := s.stor.Add(point)
	s.invalidateCache(point.BucketId)
	return p
}

func (s *Service) Update(point domain.Point) domain.Point {
	p := s.stor.Update(point.PointId, point)
	s.invalidateCache(point.BucketId)
	return p
}

func (s *Service) Remove(controlId, pointId uint64) {
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
			s.invalidateCache(point.BucketId)
			break
		}
	}
}

func (s *Service) GetByBacketId(controlId, backetId uint64) []domain.Point {
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

func (s *Service) GetByExternalIds(backetId uint64, extId []string) []domain.Point {
	if backetId == 0 || len(extId) == 0 {
		return []domain.Point{}
	}
	finds := make([]domain.Point, 0, len(extId))
	not := make([]string, 0, len(extId))

	s.rw.RLock()
	bucketCache, ok := s.cache.Get(backetId)
	s.rw.RUnlock()

	if !ok || bucketCache == nil {
		points := s.stor.GetByExternalIds(backetId, extId)
		s.fillCache(backetId, points)
		return points
	}

	for _, _extId := range extId {
		if pp, ok := bucketCache[_extId]; ok {
			finds = append(finds, *pp)
		} else {
			not = append(not, _extId)
		}
	}

	if len(not) > 0 {
		points := s.stor.GetByExternalIds(backetId, not)
		if len(points) > 0 {
			s.fillCache(backetId, points)
			finds = append(finds, points...)
		}
	}
	return finds
}

func (s *Service) GetByExternalId(backetId uint64, extId string) domain.Point {
	if backetId == 0 || extId == "" {
		return domain.Point{}
	}
	s.rw.RLock()
	bucketCache, ok := s.cache.Get(backetId)
	if ok && bucketCache != nil && bucketCache[extId] != nil {
		defer s.rw.RUnlock()
		return *bucketCache[extId]
	}
	s.rw.RUnlock()
	log.Debug(backetId, extId)
	point := s.stor.GetByExternalId([]uint64{backetId}, extId)
	log.Debug(point)
	if point != nil {
		s.fillCache(backetId, []domain.Point{*point})
		return *point
	}
	return domain.Point{}
}

func (s *Service) GetByExternal(controlId uint64, extId string) domain.Point {
	baskets, err := s.bs.GetByControl(controlId)
	if err != nil {
		return domain.Point{}
	}
	ids := make([]uint64, 0, len(baskets))
	for _, basket := range baskets {
		ids = append(ids, basket.BucketId)
	}
	point := s.stor.GetByExternalId(ids, extId)
	if point != nil {
		return *point
	}
	return domain.Point{}
}

func (s *Service) fillCache(backetId uint64, points []domain.Point) {
	if len(points) == 0 {
		return
	}
	s.rw.Lock()
	defer s.rw.Unlock()
	bucketCache, ok := s.cache.Get(backetId)
	if !ok || bucketCache == nil {
		bucketCache = make(map[string]*domain.Point)
	}
	for _, point := range points {
		bucketCache[point.ExternalId] = &point
	}
	s.cache.Set(backetId, bucketCache)
}

func (s *Service) invalidateCache(backetId uint64) {
	s.cache.Delete(backetId)
}
