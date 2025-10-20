package _map

import (
	"github.com/nx-a/ring/internal/adapter/web/server/route/dto"
	"github.com/nx-a/ring/internal/core/domain"
)

func TokenToDomain(token *dto.Token) *domain.Token {
	if token == nil {
		return nil
	}
	return &domain.Token{
		TokenId:  token.TokenId,
		BucketId: token.BucketId,
		Bucket:   token.Bucket,
		Type:     token.Type,
		Val:      token.Val,
	}
}
func TokenFromDomain(token *domain.Token) *dto.Token {
	if token == nil {
		return nil
	}
	return &dto.Token{
		TokenId:  token.TokenId,
		BucketId: token.BucketId,
		Bucket:   token.Bucket,
		Type:     token.Type,
		Val:      token.Val,
	}
}
