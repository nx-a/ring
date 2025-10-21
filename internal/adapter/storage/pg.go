package storage

import (
	"context"
	pgxLogrus "github.com/jackc/pgx-logrus"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	log "github.com/sirupsen/logrus"
	"os"
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
