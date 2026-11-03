package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	testKeyOnce sync.Once
	testKeys    [2]*rsa.PrivateKey
)

// storeKeys returns two throwaway keys, generated once per test run.
func storeKeys(t *testing.T) (*rsa.PrivateKey, *rsa.PrivateKey) {
	t.Helper()
	testKeyOnce.Do(func() {
		for i := range testKeys {
			k, err := rsa.GenerateKey(rand.Reader, 2048)
			if err != nil {
				panic(err)
			}
			testKeys[i] = k
		}
	})
	return testKeys[0], testKeys[1]
}

func TestStoreRoundTripAndModes(t *testing.T) {
	k1, _ := storeKeys(t)
	s := Store{Dir: filepath.Join(t.TempDir(), "keys")}
	if err := s.Save("a@example.com", Current, k1); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("a@example.com", Current)
	if err != nil {
		t.Fatal(err)
	}
	if !k1.Equal(got) {
		t.Error("loaded key differs from saved key")
	}

	path, _ := s.Path("a@example.com", Current)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("key file mode %v, want 0600", info.Mode().Perm())
	}
	dirInfo, _ := os.Stat(filepath.Dir(path))
	if dirInfo.Mode().Perm() != 0o700 {
		t.Errorf("key dir mode %v, want 0700", dirInfo.Mode().Perm())
	}
	data, _ := os.ReadFile(path)
	if !strings.HasPrefix(string(data), "-----BEGIN PRIVATE KEY-----") {
		t.Errorf("not PKCS #8 PEM: %.40s", data)
	}
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("leftover temp files: %v", entries)
	}
}

func TestStoreMissingIsNil(t *testing.T) {
	s := Store{Dir: t.TempDir()}
	k, err := s.Load("a@example.com", Current)
	if k != nil || err != nil {
		t.Errorf("got %v, %v; want nil, nil", k, err)
	}
}

func TestStoreRefusesReadableByOthers(t *testing.T) {
	k1, _ := storeKeys(t)
	s := Store{Dir: t.TempDir()}
	if err := s.Save("a@example.com", Current, k1); err != nil {
		t.Fatal(err)
	}
	path, _ := s.Path("a@example.com", Current)
	if err := os.Chmod(path, 0o640); err != nil {
		t.Fatal(err)
	}
	_, err := s.Load("a@example.com", Current)
	if err == nil || !strings.Contains(err.Error(), "chmod 600") {
		t.Errorf("want a chmod 600 error, got %v", err)
	}
}

// The readme's alternative: a key made outside the app (e.g. openssl genpkey), which
// writes PKCS #8 PEM. This builds the same format by hand.
func TestStoreLoadsExternallyMadeKey(t *testing.T) {
	_, k2 := storeKeys(t)
	s := Store{Dir: t.TempDir()}
	path, _ := s.Path("b@example.com", Current)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(k2)
	pemBytes := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("b@example.com", Current)
	if err != nil || !k2.Equal(got) {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestStoreRefusesOtherFormats(t *testing.T) {
	k1, _ := storeKeys(t)
	s := Store{Dir: t.TempDir()}
	path, _ := s.Path("a@example.com", Current)
	os.MkdirAll(filepath.Dir(path), 0o700)
	pkcs1 := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k1)})
	os.WriteFile(path, pkcs1, 0o600)
	if _, err := s.Load("a@example.com", Current); err == nil {
		t.Error("loaded a PKCS #1 file; want an error naming PKCS #8")
	}
	os.WriteFile(path, []byte("not a key"), 0o600)
	if _, err := s.Load("a@example.com", Current); err == nil {
		t.Error("loaded garbage")
	}
}

func TestStorePromote(t *testing.T) {
	k1, k2 := storeKeys(t)
	s := Store{Dir: t.TempDir()}
	s.Save("a@example.com", Current, k1)
	s.Save("a@example.com", Pending, k2)
	if err := s.Promote("a@example.com"); err != nil {
		t.Fatal(err)
	}
	cur, _ := s.Load("a@example.com", Current)
	pend, _ := s.Load("a@example.com", Pending)
	if !k2.Equal(cur) || pend != nil {
		t.Errorf("after promote: current is k2 = %v, pending = %v", k2.Equal(cur), pend)
	}
}

func TestStorePathStaysInsideDir(t *testing.T) {
	s := Store{Dir: "/keys"}
	p, err := s.Path("../../etc/x@example.com", Current)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(filepath.Dir(p)) != "/keys" {
		t.Errorf("path escaped the key dir: %s", p)
	}
	if _, err := s.Path("..", Current); err == nil {
		t.Error(`accepted ".." as an email`)
	}
	if _, err := s.Path("", Current); err == nil {
		t.Error("accepted an empty email")
	}
}

func TestStoreFromEnv(t *testing.T) {
	t.Setenv("DELSIGN_KEY_DIR", "")
	if got := StoreFromEnv().Dir; got != DefaultKeyDir {
		t.Errorf("default dir %q", got)
	}
	t.Setenv("DELSIGN_KEY_DIR", "/tmp/k")
	if got := StoreFromEnv().Dir; got != "/tmp/k" {
		t.Errorf("override dir %q", got)
	}
}
