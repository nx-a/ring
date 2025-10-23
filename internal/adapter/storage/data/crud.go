package data

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/dto"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type Data struct {
	pool *pgxpool.Pool
	mux  sync.Mutex
}

func New(pool *pgxpool.Pool) *Data {
	data := &Data{
		pool: pool,
	}
	return data
}
func scan(row pgx.CollectableRow) (domain.Data, error) {
	var point domain.Data
	err := row.Scan(&point.DataId, &point.PointId, &point.Time, &point.Val)
	return point, err
}

func (d *Data) Add(data []domain.Data) error {
	if data == nil {
		return fmt.Errorf("data is nil")
	}
	buckets := make(map[string][]domain.Data, 4)
	for _, l := range data {
		if buckets[l.Bucket] == nil {
			buckets[l.Bucket] = make([]domain.Data, 0, 1024)
		}
		buckets[l.Bucket] = append(buckets[l.Bucket], l)
	}
	cols := []string{"data_id", "point_id", "time", "level", "val"}
	var err error = nil
	log.Debug(buckets)
	for bucket, data := range buckets {
		_, err = d.pool.CopyFrom(context.Background(), pgx.Identifier{"data_" + bucket}, cols,
			pgx.CopyFromSlice(len(data), func(i int) ([]interface{}, error) {
				return []any{data[i].DataId, data[i].PointId, data[i].Time, data[i].Level, data[i].Val}, nil
			}))
		if err != nil {
			log.Error(err)
		}
	}
	return err
}

func (d *Data) Clear(backet string, of time.Time) {
	ct, err := d.pool.Exec(context.Background(), "DELETE FROM data_"+backet+" WHERE time < $1", of)
	if err != nil {
		log.Error(err)
		return
	}
	if ct.RowsAffected() == 0 {
		return
	}
	_, err = d.pool.Exec(context.Background(), "VACUUM data_"+backet)
	if err != nil {
		log.Error(err)
	}
}
func (d *Data) Find(data *dto.DataSelect) []domain.Data {
	//rows, err := d.pool.Query(context.Background(), "SELECT * FROM data_"+backet+" WHERE time between $1 and $2", from, to)
	return nil
}
func (d *Data) Select(backet string, from time.Time, to time.Time) []domain.Data {
	rows, err := d.pool.Query(context.Background(), "SELECT * FROM data_"+backet+" WHERE time between $1 and $2", from, to)
	if err != nil {
		log.Error(err)
		return nil
	}
	defer rows.Close()
	elements, err := pgx.CollectRows(rows, scan)
	if err != nil {
		log.Error(err)
	}
	return elements
}
func (d *Data) Create(bucket string) {
	var val int
	d.pool.QueryRow(context.Background(), "SELECT 1 from information_schema.tables where table_name = $1 and table_schema = 'public'", bucket).Scan(&val)
	log.Debug(val)
	if val == 0 {
		log.Info("Creating table ", bucket)
		sql := fmt.Sprintf("create table %s (data_id char(36) primary key, point_id bigint, time timestamp, level varchar(10), val jsonb)", "data_"+bucket)
		exec, err := d.pool.Exec(context.Background(), sql)
		if err != nil {
			log.Error(err)
			return
		}
		log.Debug(exec.RowsAffected())
		exec, err = d.pool.Exec(context.Background(), fmt.Sprintf("create index %s on %s (time, point_id)", "data_"+bucket+"_time_index", "public.data_"+bucket))
		if err != nil {
			log.Error(err)
			return
		}
		log.Debug(exec.RowsAffected())
	}
}
