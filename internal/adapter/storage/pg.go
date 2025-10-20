package storage

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	pool *pgxpool.Pool
}
type Config interface {
	Get(string) string
}

func New(cfg Config) *Storage {
	pool, err := pgxpool.New(context.Background(), cfg.Get("database.dsn"))
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
