package server

import (
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

//go:embed keys/private
var private []byte

//go:embed keys/public
var public []byte

func NewJwt(claim map[string]any) (string, error) {
	claims := jwt.MapClaims{}
	for k, v := range claim {
		claims[k] = v
	}
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(private)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	return token.SignedString(privateKey)
}

func Verify(token string) (map[string]any, error) {
	data, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		block, _ := pem.Decode(public)
		if block == nil {
			return nil, fmt.Errorf("failed to decode public key PEM")
		}
		pubKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse public key: %w", err)
		}
		pk, ok := pubKeyInterface.(*ecdsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key is not ECDSA")
		}
		return pk, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodES256.Alg()}))
	if err != nil {
		return nil, err
	}
	if claim, ok := data.Claims.(jwt.MapClaims); ok && data.Valid {
		return claim, nil
	}
	return nil, fmt.Errorf("invalid token")
}
