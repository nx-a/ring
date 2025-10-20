package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
)

func main() {
	curve := elliptic.P256()
	privateKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		log.Fatalf("Failed to generate ECDSA private key: %v", err)
	}
	derBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		fmt.Printf("Failed to marshal private key: %v\n", err)
		return
	}
	pemBlock := &pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: derBytes,
	}
	pemBytes := pem.EncodeToMemory(pemBlock)
	fmt.Printf("%s\n", pemBytes)
	publicKeyDerBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		fmt.Printf("Failed to marshal public key: %v\n", err)
		return
	}
	publicKeyPemBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyDerBytes,
	}
	publicKeyPemBytes := pem.EncodeToMemory(publicKeyPemBlock)
	fmt.Printf("%s\n", publicKeyPemBytes)

}
