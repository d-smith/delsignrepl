package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
)

func Generate() (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	publicKey := &privateKey.PublicKey
	return privateKey, publicKey
}

func Encode(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) (string, string) {
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(publicKey)
	privateKeyString := hex.EncodeToString(privateKeyBytes)
	publicKeyString := hex.EncodeToString(publicKeyBytes)
	return privateKeyString, publicKeyString
}

func Decode(privateKeyString string, publicKeyString string) (*rsa.PrivateKey, *rsa.PublicKey) {
	privateKeyBytes, _ := hex.DecodeString(privateKeyString)
	publicKeyBytes, _ := hex.DecodeString(publicKeyString)
	privateKey, _ := x509.ParsePKCS1PrivateKey(privateKeyBytes)
	publicKey, _ := x509.ParsePKIXPublicKey(publicKeyBytes)
	return privateKey, publicKey.(*rsa.PublicKey)
}

func Sign(msg string, privateKey *rsa.PrivateKey) string {
	hash := sha256.Sum256([]byte(msg))

	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		panic(err) // TODO - consider the merits of robust error handling
	}

	return hex.EncodeToString(sig)
}
