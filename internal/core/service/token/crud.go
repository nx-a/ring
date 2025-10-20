package token

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/nx-a/ring/internal/core/domain"
	"github.com/nx-a/ring/internal/core/ports"
	"github.com/nx-a/ring/internal/core/service/bucket"
)

type Token struct {
	stor          ports.TokenStorage
	bucketService *bucket.Service
}

func New(stor ports.TokenStorage, bucketService *bucket.Service) *Token {
	return &Token{stor: stor, bucketService: bucketService}
}

func (t *Token) GetByToken(token string) (map[string]any, error) {
	d, e := t.stor.GetByToken(token)
	if e != nil {
		return nil, e
	}
	_backet, err := t.bucketService.Get(d.BucketId)
	if err != nil {
		return nil, err
	}
	resp := make(map[string]any, 5)
	resp["tokenId"] = d.TokenId
	resp["bucketId"] = d.BucketId
	resp["bucket"] = _backet.SystemName
	resp["type"] = d.Type
	resp["token"] = d.Val
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
	return t.stor.Add(token)
}
func (t *Token) GetByBucketId(controlId uint64, bucketId uint64) []domain.Token {
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
func (t *Token) Remove(controlId uint64, id uint64) {
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
