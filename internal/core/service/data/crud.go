package data

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/core/service/bucket"
	appctx "github.com/nx-a/ring/internal/engine/context"
	"github.com/nx-a/ring/internal/engine/conv"
	"github.com/nx-a/ring/internal/engine/event"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	stor          ports.DataStorage
	cron          *cron.Cron
	bucketService *bucket.Service
	history       []domain.Data
	ch            chan domain.Data
	mux           sync.Mutex
	ps            ports.PointService
	loadDone      chan struct{}
	subs          map[uint64]chan domain.Data
	subMu         sync.Mutex
	subSeq        uint64
}

func New(stor ports.DataStorage, bucketService *bucket.Service, ps ports.PointService, _event *event.Subscriber) *Service {
	data := &Service{stor: stor, ps: ps,
		cron: cron.New(cron.WithSeconds()), bucketService: bucketService,
		ch:       make(chan domain.Data, 1024),
		history:  make([]domain.Data, 0, 1024),
		loadDone: make(chan struct{}),
		subs:     make(map[uint64]chan domain.Data),
	}
	_, _ = data.cron.AddFunc("1 1 * * * *", func() {
		go data.Clear()
	})
	_, err := data.cron.AddFunc("* * * * * *", func() {
		go data.sync()
	})
	if err != nil {
		log.Error(err)
	}
	_event.On("bucket", func(ctx context.Context) {
		log.Debug("on bucket event")
		data.create(appctx.BucketID(ctx))
	})
	data.cron.Start()
	go data.load()
	return data
}

func (s *Service) Find(ctx context.Context, data *dto.DataSelect) ([]domain.Data, error) {
	control, ok := appctx.Control(ctx)
	if !ok {
		return nil, fmt.Errorf("not correct token")
	}
	if _controlId, ok := control["ControlId"]; ok {
		return s.dataFind(conv.ToUint(_controlId), data)
	}
	bucketId := conv.ToUint(control["bucketId"])
	if len(data.Ext) > 0 {
		pt := s.ps.GetByExternalIds(bucketId, data.Ext)
		ids := make([]uint64, 0, len(pt))
		for _, p := range pt {
			ids = append(ids, p.PointId)
		}
		data.Points = ids
	}
	data.BucketId = bucketId
	return s.stor.Find(data), nil
}

func (s *Service) dataFind(controlId uint64, data *dto.DataSelect) ([]domain.Data, error) {
	if data.BucketId == 0 {
		return nil, fmt.Errorf("bucket is required")
	}
	buckets, err := s.bucketService.GetByControl(controlId)
	if err != nil {
		return nil, err
	}
	find := false
	for _, _bucket := range buckets {
		if _bucket.BucketId == data.BucketId {
			find = true
			break
		}
	}
	if !find {
		return nil, fmt.Errorf("not found bucket")
	}
	return s.stor.Find(data), nil
}

func (s *Service) Write(ctx context.Context, data *domain.Data) error {
	control, ok := appctx.Control(ctx)
	if !ok {
		return fmt.Errorf("not correct token")
	}
	bucketId := conv.ToUint(control["bucketId"])
	if bucketId == 0 {
		return fmt.Errorf("not correct bucket")
	}
	pt := s.ps.GetByExternalId(bucketId, data.Ext)
	if pt.PointId == 0 {
		return fmt.Errorf("point not found by external %s", data.Ext)
	}
	_uuid, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("generate uuid: %w", err)
	}
	data.DataId = _uuid.String()
	data.PointId = pt.PointId
	data.BucketId = bucketId
	if data.Level == "" {
		data.Level = "INFO"
	} else {
		data.Level = strings.ToUpper(data.Level)
	}
	if data.Time == nil {
		now := time.Now()
		data.Time = &now
	}
	data.Bucket = control["bucket"].(string)
	s.add(data)
	s.publish(data)
	return nil
}

func (s *Service) add(data *domain.Data) {
	s.ch <- *data
}

func (s *Service) publish(data *domain.Data) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for _, ch := range s.subs {
		select {
		case ch <- *data:
		default:
		}
	}
}

func (s *Service) Subscribe() (ch <-chan domain.Data, id uint64) {
	c := make(chan domain.Data, 256)
	s.subMu.Lock()
	defer s.subMu.Unlock()
	s.subSeq++
	id = s.subSeq
	s.subs[id] = c
	return c, id
}

func (s *Service) Unsubscribe(id uint64) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	if ch, ok := s.subs[id]; ok {
		delete(s.subs, id)
		close(ch)
	}
}

func (s *Service) Count(ctx context.Context, bucketId uint64) (int64, error) {
	return s.stor.Count(bucketId)
}

func (s *Service) CountAll(ctx context.Context) (map[uint64]int64, error) {
	return s.stor.CountAll()
}

const capacity = 2048

func (s *Service) sync() {
	var cs []domain.Data
	s.mux.Lock()
	if len(s.history) == 0 {
		s.mux.Unlock()
		return
	}
	cs = s.history
	newCap := max(64, min(cap(cs), capacity))
	s.history = make([]domain.Data, 0, newCap)
	s.mux.Unlock()
	if err := s.stor.Add(cs); err != nil {
		log.Error(err)
	}
}
func (s *Service) load() {
	defer close(s.loadDone)
	for dat := range s.ch {
		s.mux.Lock()
		s.history = append(s.history, dat)
		s.mux.Unlock()
	}
}
func (s *Service) Shutdown() {
	s.cron.Stop()
	close(s.ch)
	<-s.loadDone
	s.sync()
	s.subMu.Lock()
	for _, ch := range s.subs {
		close(ch)
	}
	s.subs = make(map[uint64]chan domain.Data)
	s.subMu.Unlock()
}

func (s *Service) Close() error {
	s.Shutdown()
	return nil
}
func (s *Service) create(bucketId uint64) {
	if bucketId == 0 {
		log.Error("bucket id is zero")
		return
	}
	s.stor.Create(bucketId)
}
func (s *Service) Clear() {
	lst := s.bucketService.GetAll()
	if len(lst) == 0 {
		return
	}
	now := time.Now()
	for _, _bucket := range lst {
		result := now.Add(-time.Duration(_bucket.TimeLife) * time.Hour)
		s.clearDuration(_bucket.BucketId, result)
	}
}
func (s *Service) clearDuration(bucketId uint64, of time.Time) {
	if bucketId == 0 {
		return
	}
	go s.stor.Clear(bucketId, of)
}
