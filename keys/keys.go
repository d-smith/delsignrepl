package keys

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
)

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
