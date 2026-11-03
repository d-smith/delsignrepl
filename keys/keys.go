package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
)

// Generate makes a new RSA-2048 signing key.
func Generate() (*rsa.PrivateKey, error) {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generating RSA key: %w", err)
	}
	return k, nil
}

// PublicKeyHex is the API's public key format: hex of the SPKI (PKIX) DER encoding.
func PublicKeyHex(pub *rsa.PublicKey) (string, error) {
	if pub == nil {
		return "", errors.New("no public key")
	}
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "", fmt.Errorf("encoding public key: %w", err)
	}
	return hex.EncodeToString(der), nil
}

// Encode returns hex of the private key (PKCS #1 DER) and of the public key (SPKI DER).
func Encode(privateKey *rsa.PrivateKey, publicKey *rsa.PublicKey) (string, string, error) {
	if privateKey == nil {
		return "", "", errors.New("no private key")
	}
	pub, err := PublicKeyHex(publicKey)
	if err != nil {
		return "", "", err
	}
	return hex.EncodeToString(x509.MarshalPKCS1PrivateKey(privateKey)), pub, nil
}

// Decode reverses Encode.
func Decode(privateKeyString string, publicKeyString string) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privBytes, err := hex.DecodeString(privateKeyString)
	if err != nil {
		return nil, nil, fmt.Errorf("private key hex: %w", err)
	}
	pubBytes, err := hex.DecodeString(publicKeyString)
	if err != nil {
		return nil, nil, fmt.Errorf("public key hex: %w", err)
	}
	priv, err := x509.ParsePKCS1PrivateKey(privBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("private key: %w", err)
	}
	parsed, err := x509.ParsePKIXPublicKey(pubBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("public key: %w", err)
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("public key isn't RSA")
	}
	return priv, pub, nil
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
