package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
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

// Sign returns the API's signature format: RSASSA-PKCS1-v1_5 with SHA-256 over the
// UTF-8 message, as lowercase hex.
func Sign(msg string, privateKey *rsa.PrivateKey) (string, error) {
	if privateKey == nil {
		return "", errors.New("no signing key: generate a key first")
	}
	hash := sha256.Sum256([]byte(msg))

	sig, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(sig), nil
}
