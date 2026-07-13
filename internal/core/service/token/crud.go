package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/core/service/bucket"
	"github.com/nx-a/ring/internal/engine/cache"
)

type Token struct {
	stor          ports.TokenStorage
	bucketService *bucket.Service
	cache         *cache.TTL[string, map[string]any]
}

func New(stor ports.TokenStorage, bucketService *bucket.Service) *Token {
	return &Token{
		stor:          stor,
		bucketService: bucketService,
		cache:         cache.NewTTL[string, map[string]any](5*time.Minute, 10*time.Minute),
	}
}

func (t *Token) GetByToken(token string) (map[string]any, error) {
	if cached, ok := t.cache.Get(token); ok {
		return cached, nil
	}
	d, e := t.stor.GetByToken(token)
	if e != nil {
		return nil, e
	}
	_backet, err := t.bucketService.Get(d.BucketId)
	if err != nil {
		return nil, err
	}
	resp := map[string]any{
		"tokenId":  d.TokenId,
		"bucketId": d.BucketId,
		"bucket":   _backet.SystemName,
		"type":     d.Type,
		"token":    d.Val,
	}
	t.cache.Set(token, resp)
	return resp, nil
}

func (t *Token) Add(controlId uint64, token domain.Token) domain.Token {
	buckets, err := t.bucketService.GetByControl(controlId)
	if err != nil {
		return token
	}
	find := false
	for _, _bucket := range buckets {
		if _bucket.BucketId == token.BucketId {
			find = true
			break
		}
	}
	if !find {
		return token
	}
	token.Val, err = generateRandomToken(128)
	if err != nil {
		return token
	}
	created := t.stor.Add(token)
	return created
}

func (t *Token) GetByBucketId(controlId, bucketId uint64) []domain.Token {
	buckets, err := t.bucketService.GetByControl(controlId)
	if err != nil {
		return nil
	}
	find := false
	for _, _bucket := range buckets {
		if _bucket.BucketId == bucketId {
			find = true
			break
		}
	}
	if !find {
		return nil
	}
	return t.stor.GetByBucketId(bucketId)
}

func (t *Token) Remove(controlId, id uint64) {
	_token, err := t.stor.GetById(id)
	if err != nil {
		return
	}
	buckets, err := t.bucketService.GetByControl(controlId)
	if err != nil {
		return
	}
	for _, _bucket := range buckets {
		if _bucket.BucketId == _token.BucketId {
			t.cache.Delete(_token.Val)
			t.stor.Remove(id)
			break
		}
	}
}

func generateRandomToken(length int) (string, error) {
	b := make([]byte, length)

	_, err := rand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(b), nil
}
