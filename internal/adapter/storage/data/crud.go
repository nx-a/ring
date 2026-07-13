package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
	"github.com/nx-a/ring/internal/engine/validate"
	log "github.com/sirupsen/logrus"
)

type Data struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Data {
	return &Data{pool: pool}
}

func scan(row pgx.CollectableRow) (domain.Data, error) {
	var point domain.Data
	err := row.Scan(&point.DataId, &point.BucketId, &point.PointId, &point.Time, &point.Level, &point.Val)
	return point, err
}

func (d *Data) Add(data []domain.Data) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}
	if len(data) == 0 {
		return nil
	}
	buckets := make(map[uint64][]domain.Data, 4)
	for _, l := range data {
		buckets[l.BucketId] = append(buckets[l.BucketId], l)
	}
	cols := []string{"data_id", "bucket_id", "point_id", "time", "level", "val"}

	ctx := context.Background()
	tx, err := d.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.WithError(rbErr).Error("rollback failed")
			}
		}
	}()

	for bucketId, bucketData := range buckets {
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"data"}, cols,
			pgx.CopyFromSlice(len(bucketData), func(i int) ([]interface{}, error) {
				return []any{bucketData[i].DataId, bucketId, bucketData[i].PointId, bucketData[i].Time, bucketData[i].Level, bucketData[i].Val}, nil
			}))
		if err != nil {
			log.WithError(err).Error("copy from data")
			return fmt.Errorf("copy from data: %w", err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func (d *Data) Clear(bucketId uint64, of time.Time) {
	ct, err := d.pool.Exec(context.Background(), "DELETE FROM data WHERE bucket_id = $1 AND time < $2", bucketId, of)
	if err != nil {
		log.WithError(err).Error("clear data")
		return
	}
	if ct.RowsAffected() == 0 {
		return
	}
	log.WithField("rows", ct.RowsAffected()).Debug("data cleared")
}

func (d *Data) Find(data *dto.DataSelect) []domain.Data {
	where := make([]string, 0, 6)
	ww := make([]any, 0, 6)
	where = append(where, "bucket_id = $1")
	ww = append(ww, data.BucketId)
	if data.TimeStart != nil {
		where = append(where, "time >= $2")
		ww = append(ww, data.TimeStart)
	}
	if data.TimeEnd != nil {
		where = append(where, fmt.Sprintf("time <= $%d", len(ww)+1))
		ww = append(ww, data.TimeEnd)
	}
	if len(data.Level) > 0 {
		where = append(where, fmt.Sprintf("level =ANY($%d)", len(ww)+1))
		ww = append(ww, data.Level)
	}
	if len(data.Points) > 0 {
		where = append(where, fmt.Sprintf("point_id =ANY($%d)", len(ww)+1))
		ww = append(ww, data.Points)
	}
	if len(data.Data) > 0 {
		where = append(where, fmt.Sprintf("val ->>ANY($%d)", len(ww)+1))
		ww = append(ww, data.Data)
	}
	_select := fmt.Sprintf("SELECT * FROM data WHERE %s", strings.Join(where, " and "))
	if data.Limit > 0 {
		_select += fmt.Sprintf(" LIMIT %d", data.Limit)
		if data.Offset > 0 {
			_select += fmt.Sprintf(" OFFSET %d", data.Offset)
		}
	}
	log.Debug(_select)
	rows, err := d.pool.Query(context.Background(), _select, ww...)
	if err != nil {
		log.WithError(err).Error("find data")
		return nil
	}
	defer rows.Close()
	elements, err := pgx.CollectRows(rows, scan)
	if err != nil {
		log.WithError(err).Error("collect rows")
	}
	return elements
}

func (d *Data) Select(bucketId uint64, from, to time.Time) []domain.Data {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM data WHERE bucket_id = $1 AND time between $2 and $3", bucketId, from, to)
	if err != nil {
		log.WithError(err).Error("select data")
		return nil
	}
	defer rows.Close()
	elements, err := pgx.CollectRows(rows, scan)
	if err != nil {
		log.WithError(err).Error("collect rows")
	}
	return elements
}

func (d *Data) Count(bucketId uint64) (int64, error) {
	var count int64
	err := d.pool.QueryRow(context.Background(), "SELECT COUNT(*) FROM data WHERE bucket_id = $1", bucketId).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (d *Data) CountAll() (map[uint64]int64, error) {
	rows, err := d.pool.Query(context.Background(), "SELECT bucket_id, COUNT(*) FROM data GROUP BY bucket_id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[uint64]int64)
	for rows.Next() {
		var bucketId uint64
		var count int64
		if err := rows.Scan(&bucketId, &count); err != nil {
			log.WithError(err).Error("scan count")
			continue
		}
		result[bucketId] = count
	}
	return result, nil
}

func (d *Data) Create(bucketId uint64) {
	if bucketId == 0 {
		log.Error("bucket id is zero")
		return
	}
	partitionName := fmt.Sprintf("data_bucket_%d", bucketId)
	if err := validate.PartitionName(partitionName); err != nil {
		log.WithError(err).Error("invalid partition name")
		return
	}
	var exists bool
	err := d.pool.QueryRow(context.Background(),
		"SELECT EXISTS (SELECT 1 FROM pg_tables WHERE tablename = $1 AND schemaname = 'public')",
		partitionName).Scan(&exists)
	if err != nil {
		log.WithError(err).Error("check partition exists")
		return
	}
	if exists {
		return
	}
	sql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s PARTITION OF data FOR VALUES IN (%d)", partitionName, bucketId)
	if _, err := d.pool.Exec(context.Background(), sql); err != nil {
		log.WithError(err).Error("create partition")
		return
	}
	log.WithField("partition", partitionName).Info("partition created")
}
