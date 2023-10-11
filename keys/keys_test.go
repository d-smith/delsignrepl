package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"reflect"
	"testing"
)

func TestKeyEncoding(t *testing.T) {
	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
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
