package bucket

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/adapter/storage"
	"github.com/nx-a/ring/internal/core/domain"
	log "github.com/sirupsen/logrus"
)

type Bucket struct {
	pool *pgxpool.Pool
	crud *storage.CRUD[domain.Bucket, uint64]
}

func New(pool *pgxpool.Pool) *Bucket {
	return &Bucket{pool: pool,
		crud: storage.NewCrud[domain.Bucket, uint64](scan, pool,
			"select bucket_id, control_id, system_name, time_life, time_zone from bucket where bucket_id = $1",
			"select bucket_id, control_id, system_name, time_life, time_zone from bucket where bucket_id in ($1)",
			"insert into bucket (control_id, system_name, time_life, time_zone) values ($1, $2, $3, $4) RETURNING bucket_id",
			"update bucket set control_id = $2, system_name = $3, time_life = $4, time_zone = $5 where bucket_id = $1",
			"delete from bucket where bucket_id = $1",
		)}
}
func scan(row pgx.CollectableRow) (domain.Bucket, error) {
	var bucket domain.Bucket
	err := row.Scan(&bucket.BucketId, &bucket.ControlId, &bucket.SystemName, &bucket.TimeLife, &bucket.TimeZone)
	return bucket, err
}
func (b *Bucket) Add(bucket domain.Bucket) domain.Bucket {
	bucket.BucketId = b.crud.Add(bucket.ControlId, bucket.SystemName, bucket.TimeLife, bucket.TimeZone)
	return bucket
}
func (b *Bucket) GetAll() []domain.Bucket {
	rows, err := b.pool.Query(context.Background(), "select distinct system_name, time_life from bucket where time_life > 0")
	if err != nil {
		log.Error("Failed to query bucket systems:", err)
		return nil
	}
	defer rows.Close()
	snames := make([]domain.Bucket, 0, len(rows.RawValues()))
	for rows.Next() {
		var el domain.Bucket
		if err := rows.Scan(&el.SystemName, &el.TimeLife); err != nil {
			log.Debug("Failed to scan system name:", err)
			continue
		}
		snames = append(snames, el)
	}
	return snames
}

func (b *Bucket) GetByIds(ids []uint64) ([]domain.Bucket, error) {
	return b.crud.GetByIds(ids)
}
func (b *Bucket) GetById(id uint64) (domain.Bucket, error) {
	return b.crud.GetById(id)
}
func (b *Bucket) GetByControlId(id uint64) ([]domain.Bucket, error) {
	rows, err := b.pool.Query(context.Background(), "select * from bucket where control_id = $1", id)
	if err != nil {
		return nil, err
	}
	elements, err := pgx.CollectRows(rows, scan)
	return elements, err
}
func (b *Bucket) Update(id uint64, backet domain.Bucket) domain.Bucket {
	rows := b.crud.Update(id, backet.SystemName, backet.TimeLife, backet.TimeZone)
	log.Info(rows)
	backet.BucketId = id
	return backet
}
func (b *Bucket) Remove(id uint64) {
	rows := b.crud.Delete(id)
	log.Info(rows)
}
