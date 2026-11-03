package keys_test

// The register/rotate flow (plan D2) against a fake backend written from openapi.yaml's
// /keyreg and /users/{email} descriptions. Signatures are checked with the standard
// library, not backend code.

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"delsignrepl/api"
	"delsignrepl/keys"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const email = "a@example.com"

type fakeBackend struct {
	t            *testing.T
	registered   string // pubkey hex, "" if none
	gets, posts  int
	conflicts    int  // answer this many keyregs with 409 first
	dropResponse bool // apply the keyreg, then close the connection without answering
	lastBody     map[string]any
}

func verify(pubHex, msg, sigHex string) bool {
	der, err := hex.DecodeString(pubHex)
	if err != nil {
		return false
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return false
	}
	pub, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return false
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return false
	}
	h := sha256.Sum256([]byte(msg))
	return rsa.VerifyPKCS1v15(pub, crypto.SHA256, h[:], sig) == nil
}

func (f *fakeBackend) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET" && r.URL.Path == "/api/v1/users/"+email:
		f.gets++
		if f.registered == "" {
			http.Error(w, "No key registered", 404)
			return
		}
		json.NewEncoder(w).Encode(api.UserInfo{Email: email, PubKey: f.registered})

	case r.Method == "POST" && r.URL.Path == "/api/v1/keyreg":
		f.posts++
		var raw bytes.Buffer
		raw.ReadFrom(r.Body)
		json.Unmarshal(raw.Bytes(), &f.lastBody)
		d := json.NewDecoder(bytes.NewReader(raw.Bytes()))
		d.DisallowUnknownFields()
		var reg api.KeyReg
		if err := d.Decode(&reg); err != nil || reg.Email != email {
			http.Error(w, "bad body", 400)
			return
		}
		if !verify(reg.PubKey, reg.Email, reg.SignatureForRegistration) {
			http.Error(w, "sig4reg doesn't verify", 400)
			return
		}
		if f.registered != "" && !verify(f.registered, reg.Email+reg.PubKey, reg.Sig4Rotate) {
			http.Error(w, "A key is already registered; replacing it needs sig4rotate from the current key", 403)
			return
		}
		if f.conflicts > 0 {
			f.conflicts--
			http.Error(w, "key changed", 409)
			return
		}
		f.registered = reg.PubKey
		if f.dropResponse {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				f.t.Fatal(err)
			}
			conn.Close()
		}

	default:
		f.t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		http.Error(w, "not found", 404)
	}
}

type fixture struct {
	backend    *fakeBackend
	client     *api.Client
	store      keys.Store
	key1, key2 *rsa.PrivateKey
	hex1, hex2 string
}

func setup(t *testing.T) *fixture {
	t.Helper()
	vf, privs := loadVectors(t)
	f := &fixture{backend: &fakeBackend{t: t}, store: keys.Store{Dir: t.TempDir()},
		key1: privs["key1"], key2: privs["key2"],
		hex1: vf.Keys["key1"].PublicKeySpkiHex, hex2: vf.Keys["key2"].PublicKeySpkiHex}
	srv := httptest.NewServer(f.backend)
	t.Cleanup(srv.Close)
	f.client = &api.Client{BaseURL: srv.URL, Token: "tok", HTTP: srv.Client()}
	return f
}

func (f *fixture) save(t *testing.T, name string, k *rsa.PrivateKey) {
	t.Helper()
	if err := f.store.Save(email, name, k); err != nil {
		t.Fatal(err)
	}
}

func (f *fixture) load(t *testing.T, name string) *rsa.PrivateKey {
	t.Helper()
	k, err := f.store.Load(email, name)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestRegisterFirstKey(t *testing.T) {
	f := setup(t)
	f.save(t, keys.Current, f.key1)

	out, err := keys.Register(f.client, f.store, email)
	if err != nil || out != keys.Registered {
		t.Fatalf("got %v, %v", out, err)
	}
	if f.backend.registered != f.hex1 {
		t.Error("backend didn't register key1")
	}
	if _, ok := f.backend.lastBody["sig4rotate"]; ok {
		t.Error("first registration sent sig4rotate")
	}
}

func TestRegisterRotates(t *testing.T) {
	f := setup(t)
	f.backend.registered = f.hex1
	f.save(t, keys.Current, f.key1)
	f.save(t, keys.Pending, f.key2)

	out, err := keys.Register(f.client, f.store, email)
	if err != nil || out != keys.Rotated {
		t.Fatalf("got %v, %v", out, err)
	}
	if f.backend.registered != f.hex2 {
		t.Error("backend didn't switch to key2")
	}
	if !f.key2.Equal(f.load(t, keys.Current)) || f.load(t, keys.Pending) != nil {
		t.Error("store didn't promote key2")
	}
}

// The rotation reaches the backend but its response is lost. The pending key was on
// disk all along, so the next Register finds it registered and promotes it.
func TestRegisterLostRotationResponseThenPromotes(t *testing.T) {
	f := setup(t)
	f.backend.registered = f.hex1
	f.backend.dropResponse = true
	f.save(t, keys.Current, f.key1)
	f.save(t, keys.Pending, f.key2)

	_, err := keys.Register(f.client, f.store, email)
	var netErr *api.NetworkError
	if !errors.As(err, &netErr) {
		t.Fatalf("want a NetworkError, got %T %v", err, err)
	}
	if f.backend.registered != f.hex2 {
		t.Fatal("fake backend should have applied the rotation")
	}
	if !f.key1.Equal(f.load(t, keys.Current)) || !f.key2.Equal(f.load(t, keys.Pending)) {
		t.Fatal("store changed although the response was lost")
	}

	if st, err := keys.CheckKey(f.client, f.store, email); err != nil || st != keys.FinishRotation {
		t.Errorf("CheckKey: got %v, %v; want FinishRotation", st, err)
	}

	f.backend.dropResponse = false
	posts := f.backend.posts
	out, err := keys.Register(f.client, f.store, email)
	if err != nil || out != keys.Promoted {
		t.Fatalf("got %v, %v", out, err)
	}
	if f.backend.posts != posts {
		t.Error("promoting shouldn't post to keyreg")
	}
	if !f.key2.Equal(f.load(t, keys.Current)) {
		t.Error("key2 isn't current after promote")
	}
}

func TestRegisterMismatchDoesNothing(t *testing.T) {
	f := setup(t)
	f.backend.registered = f.hex2
	f.save(t, keys.Current, f.key1)

	_, err := keys.Register(f.client, f.store, email)
	if !errors.Is(err, keys.ErrKeyMismatch) {
		t.Fatalf("want ErrKeyMismatch, got %v", err)
	}
	if f.backend.posts != 0 {
		t.Error("a mismatch shouldn't post to keyreg")
	}
}

func TestRegisterAlreadyRegistered(t *testing.T) {
	f := setup(t)
	f.backend.registered = f.hex1
	f.save(t, keys.Current, f.key1)

	out, err := keys.Register(f.client, f.store, email)
	if err != nil || out != keys.AlreadyRegistered || f.backend.posts != 0 {
		t.Fatalf("got %v, %v, %d posts", out, err, f.backend.posts)
	}
}

func TestRegister409ReloadsAndRetriesOnce(t *testing.T) {
	f := setup(t)
	f.backend.conflicts = 1
	f.save(t, keys.Current, f.key1)

	out, err := keys.Register(f.client, f.store, email)
	if err != nil || out != keys.Registered {
		t.Fatalf("got %v, %v", out, err)
	}
	if f.backend.gets != 2 || f.backend.posts != 2 {
		t.Errorf("want 2 GETs and 2 POSTs, got %d and %d", f.backend.gets, f.backend.posts)
	}
}

func TestRegister409TwiceGivesUp(t *testing.T) {
	f := setup(t)
	f.backend.conflicts = 5
	f.save(t, keys.Current, f.key1)

	_, err := keys.Register(f.client, f.store, email)
	if api.StatusOf(err) != 409 || f.backend.posts != 2 {
		t.Fatalf("want a 409 after 2 posts, got %v after %d", err, f.backend.posts)
	}
}

func TestRegisterNoKey(t *testing.T) {
	f := setup(t)
	if _, err := keys.Register(f.client, f.store, email); !errors.Is(err, keys.ErrNoKey) {
		t.Fatalf("want ErrNoKey, got %v", err)
	}
}

func TestCheckKey(t *testing.T) {
	cases := []struct {
		name       string
		registered func(f *fixture) string
		current    bool
		pending    bool
		want       keys.Status
	}{
		{"no local key", func(f *fixture) string { return "" }, false, false, keys.NoLocalKey},
		{"not registered", func(f *fixture) string { return "" }, true, false, keys.NotRegistered},
		{"matches", func(f *fixture) string { return f.hex1 }, true, false, keys.Matches},
		{"matches, uppercase hex", func(f *fixture) string { return strings.ToUpper(f.hex1) }, true, false, keys.Matches},
		{"pending ready", func(f *fixture) string { return f.hex1 }, true, true, keys.PendingReady},
		{"mismatch", func(f *fixture) string { return f.hex2 }, true, false, keys.Mismatch},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := setup(t)
			f.backend.registered = tc.registered(f)
			if tc.current {
				f.save(t, keys.Current, f.key1)
			}
			if tc.pending {
				f.save(t, keys.Pending, f.key2)
			}
			st, err := keys.CheckKey(f.client, f.store, email)
			if err != nil || st != tc.want {
				t.Errorf("got %v, %v; want %v", st, err, tc.want)
			}
			if f.backend.posts != 0 {
				t.Error("CheckKey must not change anything")
			}
		})
	}
}
