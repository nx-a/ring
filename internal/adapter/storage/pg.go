package storage

import (
	"context"
	"os"
	"time"

	pgxLogrus "github.com/jackc/pgx-logrus"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/nx-a/ring/internal/engine/conv"
	log "github.com/sirupsen/logrus"
)

type Storage struct {
	pool *pgxpool.Pool
}

type Config interface {
	Get(string) string
}

func New(cfg Config) *Storage {
	config, err := pgxpool.ParseConfig(cfg.Get("database.dsn"))
	if err != nil {
		log.Fatal(err)
	}
	// Pool tuning
	config.MaxConns = int32(conv.ParseInt(cfg.Get("database.maxConns"), "25"))
	config.MinConns = int32(conv.ParseInt(cfg.Get("database.minConns"), "5"))
	config.MaxConnLifetime = time.Duration(conv.ParseInt(cfg.Get("database.maxConnLifetimeSec"), "3600")) * time.Second
	config.MaxConnIdleTime = time.Duration(conv.ParseInt(cfg.Get("database.maxConnIdleTimeSec"), "600")) * time.Second
	config.HealthCheckPeriod = time.Duration(conv.ParseInt(cfg.Get("database.healthCheckPeriodSec"), "60")) * time.Second

	logger := pgxLogrus.NewLogger(&log.Logger{
		Out:          os.Stderr,
		Formatter:    new(log.TextFormatter),
		Hooks:        make(log.LevelHooks),
		Level:        log.ErrorLevel,
		ExitFunc:     os.Exit,
		ReportCaller: false,
	})
	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   logger,
		LogLevel: tracelog.LogLevelTrace,
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		panic(err)
	}
	return &Storage{
		pool: pool,
	}
}
func (s *Storage) Get() *pgxpool.Pool {
	return s.pool
}
func (s *Storage) Close() error {
	s.pool.Close()
	return nil
}
