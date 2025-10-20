package storage

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
)

type CRUD[T any, I any] struct {
	scan   func(row pgx.CollectableRow) (T, error)
	pool   *pgxpool.Pool
	sel    string
	selIds string
	ins    string
	upd    string
	del    string
}

func NewCrud[T any, I any](scan func(row pgx.CollectableRow) (T, error), pool *pgxpool.Pool, sel, selIds, ins, upd, del string) *CRUD[T, I] {
	return &CRUD[T, I]{
		scan:   scan,
		pool:   pool,
		sel:    sel,
		selIds: selIds,
		ins:    ins,
		upd:    upd,
		del:    del,
	}
}
func (c CRUD[T, I]) GetByIds(id []uint64) ([]T, error) {
	rows, err := c.pool.Query(context.Background(), c.selIds, id)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer rows.Close()
	elements, err := pgx.CollectRows(rows, c.scan)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	if len(elements) > 0 {
		return elements, nil
	}
	return nil, nil

}
func (c CRUD[T, I]) GetById(id I) (T, error) {
	rows, err := c.pool.Query(context.Background(), c.sel, id)
	if err != nil {
		var t T
		log.Error(err)
		return t, err
	}
	defer rows.Close()
	elements, err := pgx.CollectRows(rows, c.scan)
	if err != nil {
		var t T
		log.Error(err)
		return t, err
	}
	if len(elements) > 0 {
		return elements[0], nil
	}
	var t T
	return t, nil

}
func (c CRUD[T, I]) Add(attr ...any) I {
	var id I
	err := c.pool.QueryRow(context.Background(), c.ins, attr...).Scan(&id)
	if err != nil {
		log.Error(err)
	}
	return id
}

func (c CRUD[T, I]) Update(attr ...any) int64 {
	rows, err := c.pool.Exec(context.Background(), c.upd, attr...)
	if err != nil {
		log.Error(err)
		return 0
	}
	return rows.RowsAffected()
}
func (c CRUD[T, I]) Delete(attr ...any) int64 {
	rows, err := c.pool.Exec(context.Background(), c.del, attr...)
	if err != nil {
		log.Error(err)
		return 0
	}
	return rows.RowsAffected()
}
