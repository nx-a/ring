package ports

import "github.com/nx-a/ring/internal/core/domain"

type TokenService interface {
	GetByToken(token string) (map[string]any, error)
	Add(controlId uint64, token domain.Token) domain.Token
	GetByBucketId(controlId uint64, id uint64) []domain.Token
	Remove(controlId uint64, id uint64)
}
