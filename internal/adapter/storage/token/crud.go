package token

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nx-a/ring/internal/adapter/storage"
	"github.com/nx-a/ring/internal/core/domain"
	log "github.com/sirupsen/logrus"
)

type Token struct {
	pool *pgxpool.Pool
	crud *storage.CRUD[domain.Token, uint64]
}

func New(pool *pgxpool.Pool) *Token {
	return &Token{pool: pool,
		crud: storage.NewCrud[domain.Token, uint64](scan, pool,
			"select token_id, backet_id, type, val from token where token_id = $1",
			"select token_id, backet_id, type, val from token where point_id in ($1)",
			"insert into token (backet_id, type, val) values ($1, $2, $3) RETURNING token_id",
			"update token set backet_id = $2, type = $3, val = $4 where token_id = $1",
			"delete from token where token_id = $1",
		)}
}
func scan(row pgx.CollectableRow) (domain.Token, error) {
	var token domain.Token
	err := row.Scan(&token.TokenId, &token.BucketId, &token.Type, &token.Val)
	return token, err
}
func (b *Token) Add(token domain.Token) domain.Token {
	token.TokenId = b.crud.Add(token.BucketId, token.Type, token.Val)
	return token
}
func (b *Token) GetByIds(ids []uint64) ([]domain.Token, error) {
	return b.crud.GetByIds(ids)
}
func (b *Token) GetById(id uint64) (domain.Token, error) {
	return b.crud.GetById(id)
}
func (b *Token) Update(id uint64, token domain.Token) domain.Token {
	rows := b.crud.Update(id, token.BucketId, token.Type, token.Val)
	log.Info(rows)
	token.TokenId = id
	return token
}
func (b *Token) Remove(id uint64) {
	rows := b.crud.Delete(id)
	log.Info(rows)
}
func (b *Token) GetByToken(val string) (domain.Token, error) {
	rows, err := b.pool.Query(context.Background(), "SELECT token_id, backet_id, type, val FROM token WHERE val = $1", val)
	if err != nil {
		log.Error(err)
		return domain.Token{}, nil
	}
	defer rows.Close()
	elements, err := pgx.CollectRows(rows, scan)
	if err != nil {
		log.Error(err)
	}
	if len(elements) != 1 {
		return domain.Token{}, fmt.Errorf("expected 1 element, got %d", len(elements))
	}
	return elements[0], nil
}
func (b *Token) GetByBucketId(id uint64) []domain.Token {
	rows, err := b.pool.Query(context.Background(), "SELECT token_id, backet_id, type, val FROM token WHERE backet_id = $1", id)
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
