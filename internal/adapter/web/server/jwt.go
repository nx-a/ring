package server

import (
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	log "github.com/sirupsen/logrus"
	"time"
)

//go:embed keys/private
var private []byte

//go:embed keys/public
var public []byte

func NewJwt(claim map[string]any) (string, error) {
	claim["exp"] = time.Now().Add(time.Hour * 24).Unix()
	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims(claim))
	privateKey, err := jwt.ParseECPrivateKeyFromPEM(private)
	log.Info(err)
	return token.SignedString(privateKey)
}
func Verify(token string) (map[string]any, error) {
	data, err := jwt.Parse(token, func(token *jwt.Token) (any, error) {
		block, _ := pem.Decode(public)
		pubKeyInterface, _ := x509.ParsePKIXPublicKey(block.Bytes)
		pk, _ := pubKeyInterface.(*ecdsa.PublicKey)
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
