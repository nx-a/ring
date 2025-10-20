package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/core/service/bucket"
	"github.com/nx-a/ring/internal/engine/event"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Service struct {
	stor          ports.DataStorage
	cron          *cron.Cron
	bucketService *bucket.Service
	history       []domain.Data
	ch            chan domain.Data
	mux           sync.Mutex
	ps            ports.PointService
}

func New(stor ports.DataStorage, bucketService *bucket.Service, ps ports.PointService, _event *event.Subscriber) *Service {
	data := &Service{stor: stor, ps: ps,
		cron: cron.New(cron.WithSeconds()), bucketService: bucketService,
		ch:      make(chan domain.Data, 1024),
		history: make([]domain.Data, 0, 1024),
	}
	_, err := data.cron.AddFunc("* * * * * *", func() {
		go data.Clear()
	})
	_, err = data.cron.AddFunc("*/1 * * * * *", func() {
		go data.sync()
	})
	if err != nil {
		log.Error(err)
	}
	_event.On("bucket", func(ctx context.Context) {
		log.Debug("on bucket event")
		data.create(ctx.Value("sysname").(string))
	})
	go data.load()
	return data
}
func (s *Service) Write(ctx context.Context, data domain.Data) {
	control, ok := ctx.Value("control").(map[string]any)
	if !ok {
		return
	}
	_bucket, ok := control["bucket"].(string)
	if !ok {
		return
	}
	pt := s.ps.GetByExternalId(control["bucketId"].(uint64), data.Ext)
	_uuid, err := uuid.NewV7()
	if err != nil {
		_uuid, _ = uuid.NewV7()
	}
	data.DataId = _uuid.String()
	data.PointId = pt.PointId
	data.Time = time.Now()
	data.Bucket = _bucket
	s.add(data)
}
func (s *Service) add(data domain.Data) {
	s.ch <- data
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
	for dat := range s.ch {
		s.mux.Lock()
		s.history = append(s.history, dat)
		s.mux.Unlock()
	}
}
func (s *Service) create(bucket string) {
	s.stor.Create(bucket)
}
func (s *Service) Clear() {
	lst := s.bucketService.GetAll()
	if lst == nil || len(lst) == 0 {
		return
	}
	now := time.Now()
	for _, _bucket := range lst {
		result := now.Add(-time.Duration(_bucket.TimeLife) * time.Hour)
		s.clearDuration(_bucket.SystemName, result)
	}
}
func (s *Service) clearDuration(bucket string, of time.Time) {
	go s.stor.Clear(bucket, of)
}
