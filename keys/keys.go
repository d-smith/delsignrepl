package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
)

func Generate() (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey
}

func Encode(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (string, string) {
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	privateKeyString := hex.EncodeToString(privateKeyBytes)
	publicKeyString := hex.EncodeToString(publicKeyBytes)
	return privateKeyString, publicKeyString
}

func Decode(privateKeyString string, publicKeyString string) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKeyBytes, _ := hex.DecodeString(privateKeyString)
	publicKeyBytes, _ := hex.DecodeString(publicKeyString)
	privateKey, _ := x509.ParseECPrivateKey(privateKeyBytes)
	publicKey, _ := x509.ParsePKIXPublicKey(publicKeyBytes)
	return privateKey, publicKey.(*ecdsa.PublicKey)
}

func Sign(msg string, privateKey *ecdsa.PrivateKey) string {
	hash := sha256.Sum256([]byte(msg))

	sig, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		panic(err) // TODO - consider the merits of robust error handling
	}

	return hex.EncodeToString(sig)
}
