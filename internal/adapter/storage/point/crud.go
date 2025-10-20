package backet

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/adapter/storage"
	"github.com/nx-a/ring/internal/core/domain"
	log "github.com/sirupsen/logrus"
)

type Point struct {
	pool *pgxpool.Pool
	crud *storage.CRUD[domain.Point, uint64]
}

func New(pool *pgxpool.Pool) *Point {
	return &Point{pool: pool,
		crud: storage.NewCrud[domain.Point, uint64](scan, pool,
			"select point_id, backet_id, external_id, time_zone from point where point_id = $1",
			"select point_id, backet_id, external_id, time_zone from point where point_id in ($1)",
			"insert into point (backet_id, external_id, time_zone) values ($1, $2, $3) RETURNING point_id",
			"update point set backet_id = $2, external_id = $3, time_zone = $4 where point_id = $1",
			"delete from point where point_id = $1",
		)}
}
func scan(row pgx.CollectableRow) (domain.Point, error) {
	var point domain.Point
	err := row.Scan(&point.PointId, &point.BacketId, &point.ExternalId, &point.TimeZone)
	return point, err
}
func (b *Point) Add(point domain.Point) domain.Point {
	point.PointId = b.crud.Add(point.BacketId, point.ExternalId, point.TimeZone)
	return point
}
func (b *Point) GetByIds(ids []uint64) ([]domain.Point, error) {
	return b.crud.GetByIds(ids)
}
func (b *Point) GetById(id uint64) (domain.Point, error) {
	return b.crud.GetById(id)
}
func (b *Point) Update(id uint64, point domain.Point) domain.Point {
	rows := b.crud.Update(id, point.BacketId, point.ExternalId, point.TimeZone)
	log.Info(rows)
	point.PointId = id
	return point
}
func (b *Point) Remove(id uint64) {
	rows := b.crud.Delete(id)
	log.Info(rows)
}
func (b *Point) GetByBacketId(id uint64) []domain.Point {
	rows, err := b.pool.Query(context.Background(), "select point_id, backet_id, external_id, time_zone from point where backet_id = $1", id)
	if err != nil {
		log.Error(err)
		return nil
	}
	defer rows.Close()
	var points []domain.Point
	for rows.Next() {
		var point domain.Point
		err := rows.Scan(&point.PointId, &point.BacketId, &point.ExternalId, &point.TimeZone)
		if err != nil {
			log.Error(err)
		}
	}
	return points
}
func (b *Point) GetByExternalId(ids []uint64, ext string) *domain.Point {
	rows, err := b.pool.Query(context.Background(), "select point_id, backet_id, external_id, time_zone from point where backet_id in($1) and external_id= $2", ids, ext)
	if err != nil {
		log.Error(err)
		return nil
	}
	defer rows.Close()
	for rows.Next() {
		var point domain.Point
		err := rows.Scan(&point.PointId, &point.BacketId, &point.ExternalId, &point.TimeZone)
		if err != nil {
			log.Error(err)
		}
		return &point
	}
	return nil
}
