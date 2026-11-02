package keys_test

// Signing must reproduce docs/delsign-api/test-vectors.json exactly (see CLAUDE.md and
// client-guide.md, "Test against the vectors first"). PKCS #1 v1.5 signatures are
// deterministic, so exact equality is the right check. The vector keys are published test
// keys: never use them outside tests.

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"delsignrepl/api"
	"delsignrepl/keys"
	"delsignrepl/send"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"testing"
	"time"
)

const vectorsPath = "../docs/delsign-api/test-vectors.json"

type vectorFile struct {
	Keys map[string]struct {
		PublicKeySpkiHex   string `json:"publicKeySpkiHex"`
		PrivateKeyPkcs8Hex string `json:"privateKeyPkcs8Hex"`
	} `json:"keys"`
	Vectors []struct {
		Name       string          `json:"name"`
		Request    string          `json:"request"`
		Body       json.RawMessage `json:"body"`
		Signatures []struct {
			Field     string `json:"field"`
			Key       string `json:"key"`
			Message   string `json:"message"`
			Signature string `json:"signature"`
		} `json:"signatures"`
	} `json:"vectors"`
}

func loadVectors(t *testing.T) (vectorFile, map[string]*rsa.PrivateKey) {
	t.Helper()
	raw, err := os.ReadFile(vectorsPath)
	if err != nil {
		t.Fatalf("reading vectors: %v", err)
	}
	var vf vectorFile
	if err := json.Unmarshal(raw, &vf); err != nil {
		t.Fatalf("parsing vectors: %v", err)
	}
	if len(vf.Vectors) == 0 || len(vf.Keys) == 0 {
		t.Fatal("vectors file has no keys or vectors")
	}
	privs := map[string]*rsa.PrivateKey{}
	for name, k := range vf.Keys {
		der, err := hex.DecodeString(k.PrivateKeyPkcs8Hex)
		if err != nil {
			t.Fatalf("%s: private key hex: %v", name, err)
		}
		parsed, err := x509.ParsePKCS8PrivateKey(der)
		if err != nil {
			t.Fatalf("%s: PKCS #8: %v", name, err)
		}
		priv, ok := parsed.(*rsa.PrivateKey)
		if !ok {
			t.Fatalf("%s: not an RSA key", name)
		}
		privs[name] = priv
	}
	return vf, privs
}

// object decodes JSON into a generic value, keeping numbers exact, so two encodings can be
// compared as JSON objects (field order and whitespace don't matter; types do).
func object(t *testing.T, b []byte) map[string]any {
	t.Helper()
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	var m map[string]any
	if err := d.Decode(&m); err != nil {
		t.Fatalf("decoding %s: %v", b, err)
	}
	return m
}

func sameJSON(t *testing.T, what string, got, want []byte) {
	t.Helper()
	if g, w := object(t, got), object(t, want); !reflect.DeepEqual(g, w) {
		t.Errorf("%s:\n got  %s\n want %s", what, got, want)
	}
}

// Checks 1-3: the RSA primitive, public key encoding, and the bodies' signature fields.
func TestVectorSignatures(t *testing.T) {
	vf, privs := loadVectors(t)

	for name, k := range vf.Keys {
		priv := privs[name]
		_, pubHex := keys.Encode(priv, &priv.PublicKey)
		if pubHex != k.PublicKeySpkiHex {
			t.Errorf("%s: public key hex differs from publicKeySpkiHex", name)
		}
	}

	for _, v := range vf.Vectors {
		body := object(t, v.Body)
		for _, s := range v.Signatures {
			t.Run(v.Name+"/"+s.Field, func(t *testing.T) {
				got, err := keys.Sign(s.Message, privs[s.Key])
				if err != nil {
					t.Fatal(err)
				}
				if got != s.Signature {
					t.Errorf("signature differs\n got  %s\n want %s", got, s.Signature)
				}
				if body[s.Field] != s.Signature {
					t.Errorf("body[%q] isn't the listed signature", s.Field)
				}
			})
		}
	}
}

// Check 4: send bodies decode into api.SendRequest with no unknown fields, the message is
// built from the body's exact strings, and re-encoding gives the same object (amount a
// string, expiry a number).
func TestVectorSendMessage(t *testing.T) {
	vf, _ := loadVectors(t)
	n := 0
	for _, v := range vf.Vectors {
		for _, s := range v.Signatures {
			if s.Field != "sig" {
				continue
			}
			n++
			t.Run(v.Name, func(t *testing.T) {
				d := json.NewDecoder(bytes.NewReader(v.Body))
				d.DisallowUnknownFields()
				var req api.SendRequest
				if err := d.Decode(&req); err != nil {
					t.Fatalf("body doesn't fit api.SendRequest: %v", err)
				}
				if got := api.SendMessage(req); got != s.Message {
					t.Errorf("SendMessage\n got  %q\n want %q", got, s.Message)
				}
				out, err := json.Marshal(req)
				if err != nil {
					t.Fatal(err)
				}
				sameJSON(t, "re-encoded body", out, v.Body)
			})
		}
	}
	if n == 0 {
		t.Fatal("no send vectors found")
	}
}

// Checks 5 and 6: keyreg bodies. sig4rotate signs email + new pubkey hex with the current
// key; BuildKeyReg reproduces the register and rotate bodies exactly.
func TestVectorKeyReg(t *testing.T) {
	vf, privs := loadVectors(t)
	cases := map[string]struct{ next, current string }{
		"register": {next: "key1"},
		"rotate":   {next: "key2", current: "key1"},
	}
	seen := 0
	for _, v := range vf.Vectors {
		c, ok := cases[v.Name]
		if !ok {
			continue
		}
		seen++
		t.Run(v.Name, func(t *testing.T) {
			var body struct {
				Email  string `json:"email"`
				Pubkey string `json:"pubkey"`
			}
			if err := json.Unmarshal(v.Body, &body); err != nil {
				t.Fatal(err)
			}
			for _, s := range v.Signatures {
				if s.Field == "sig4rotate" && s.Message != body.Email+body.Pubkey {
					t.Errorf("sig4rotate message isn't email + pubkey")
				}
			}

			var current *rsa.PrivateKey
			if c.current != "" {
				current = privs[c.current]
			}
			reg, err := keys.BuildKeyReg(body.Email, privs[c.next], current)
			if err != nil {
				t.Fatalf("BuildKeyReg: %v", err)
			}
			out, err := json.Marshal(reg)
			if err != nil {
				t.Fatal(err)
			}
			sameJSON(t, "keyreg body", out, v.Body)
		})
	}
	if seen != len(cases) {
		t.Fatalf("found %d of %d keyreg vectors", seen, len(cases))
	}
}

// Check 7: BuildSend, given the vector's clock and request id, reproduces each send body.
func TestVectorBuildSend(t *testing.T) {
	vf, privs := loadVectors(t)
	n := 0
	for _, v := range vf.Vectors {
		for _, s := range v.Signatures {
			if s.Field != "sig" {
				continue
			}
			n++
			t.Run(v.Name, func(t *testing.T) {
				var want api.SendRequest
				if err := json.Unmarshal(v.Body, &want); err != nil {
					t.Fatal(err)
				}
				wei, ok := new(big.Int).SetString(want.Amount, 10)
				if !ok {
					t.Fatalf("vector amount %q isn't decimal", want.Amount)
				}
				now := time.Unix(want.Expiry-120, 0)
				newID := func() (string, error) { return want.RequestID, nil }

				req, out, err := send.BuildSend(privs[s.Key], want.Source, want.Dest, wei, now, newID)
				if err != nil {
					t.Fatalf("BuildSend: %v", err)
				}
				if req != want {
					t.Errorf("request\n got  %+v\n want %+v", req, want)
				}
				sameJSON(t, "send body", out, v.Body)
			})
		}
	}
	if n == 0 {
		t.Fatal("no send vectors found")
	}
}
