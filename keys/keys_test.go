package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"reflect"
	"testing"
)

func TestKeyEncoding(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	priv, pub, err := Encode(privateKey, &privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	priv2, pub2, err := Decode(priv, pub)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(privateKey, priv2) {
		t.Error("private keys do not match")
	}
	if !reflect.DeepEqual(&privateKey.PublicKey, pub2) {
		t.Error("public keys do not match")
	}
	if _, _, err := Decode("zz", pub); err == nil {
		t.Error("Decode accepted bad hex")
	}
	if _, _, err := Encode(nil, nil); err == nil {
		t.Error("Encode accepted a nil key")
	}
}

func TestSignRefusesNilKey(t *testing.T) {
	if _, err := Sign("x", nil); err == nil {
		t.Error("Sign accepted a nil key")
	}
}
