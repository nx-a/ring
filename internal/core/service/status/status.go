package status

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/core/ports"
)

type Service struct {
	pool        *pgxpool.Pool
	dataService ports.DataService
	tcpStats    TCPStats
	startTime   int64
}

type TCPStats interface {
	ClientsCount() int
}

func New(pool *pgxpool.Pool, dataService ports.DataService, tcpStats TCPStats, startTime int64) *Service {
	return &Service{
		pool:        pool,
		dataService: dataService,
		tcpStats:    tcpStats,
		startTime:   startTime,
	}
}

func (s *Service) Status(ctx context.Context) (map[string]any, error) {
	stats := s.pool.Stat()
	return map[string]any{
		"db": map[string]any{
			"totalConns":    stats.TotalConns(),
			"idleConns":     stats.IdleConns(),
			"acquiredConns": stats.AcquiredConns(),
			"maxConns":      stats.MaxConns(),
		},
		"tcp": map[string]any{
			"clients": s.tcpStats.ClientsCount(),
		},
		"startTime": s.startTime,
	}, nil
}

func (s *Service) Metrics(ctx context.Context) (map[string]any, error) {
	counts, err := s.dataService.CountAll(ctx)
	if err != nil {
		return nil, err
	}
	total := int64(0)
	for _, c := range counts {
		total += c
	}
	return map[string]any{
		"total":      total,
		"perBucket":  counts,
		"buckets":    len(counts),
		"tcpClients": s.tcpStats.ClientsCount(),
	}, nil
}
