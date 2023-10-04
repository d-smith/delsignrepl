package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"reflect"
	"testing"
)

func TestKeyEncoding(t *testing.T) {
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	publicKey := &privateKey.PublicKey
	priv, pub := Encode(privateKey, &privateKey.PublicKey)
	t.Log(priv)
	t.Log(pub)

	t.Log("orig pub key: ", publicKey)

	priv2, pub2 := Decode(priv, pub)
	t.Log("decoded pub key: ", pub2)

	if !reflect.DeepEqual(privateKey, priv2) {
		t.Fatal("Private keys do not match.")
	}
	if !reflect.DeepEqual(publicKey, pub2) {
		fmt.Println("Public keys do not match.")
	}
}
