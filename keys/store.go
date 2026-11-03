package keys

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
)

// Key file names within an email's directory (plan D1).
const (
	Current = "current" // the key in use, registered or about to be
	Pending = "pending" // a replacement waiting to be registered by rotation
)

// DefaultKeyDir is used when DELSIGN_KEY_DIR isn't set. It's relative to the working
// directory and gitignored.
var DefaultKeyDir = filepath.Join(".delsign", "keys")

// Store keeps dev signing keys as PKCS #8 PEM files, mode 0600:
// <Dir>/<email>/current.pem and pending.pem. The private key exists nowhere else, so a
// lost file means a lost key.
type Store struct {
	Dir string
}

// StoreFromEnv returns the store at DELSIGN_KEY_DIR, or DefaultKeyDir.
func StoreFromEnv() Store {
	if dir := os.Getenv("DELSIGN_KEY_DIR"); dir != "" {
		return Store{Dir: dir}
	}
	return Store{Dir: DefaultKeyDir}
}

func (s Store) emailDir(email string) (string, error) {
	name := url.PathEscape(email) // no path separators survive
	if email == "" || name == "." || name == ".." {
		return "", fmt.Errorf("can't use %q as a key directory name", email)
	}
	return filepath.Join(s.Dir, name), nil
}

// Path is where the named key file for email lives.
func (s Store) Path(email, name string) (string, error) {
	dir, err := s.emailDir(email)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".pem"), nil
}

// Load reads the named key for email. It returns (nil, nil) if there's no such file.
// A file that group or others can read is refused.
func (s Store) Load(email, name string) (*rsa.PrivateKey, error) {
	path, err := s.Path(email, name)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("key file %s can be read by other users; run: chmod 600 %s", path, path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("%s isn't a PKCS #8 PEM private key (\"BEGIN PRIVATE KEY\")", path)
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	k, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%s isn't an RSA key", path)
	}
	return k, nil
}

// Save writes the named key for email atomically (temp file, then rename), mode 0600,
// creating the directory with mode 0700.
func (s Store) Save(email, name string, k *rsa.PrivateKey) error {
	path, err := s.Path(email, name)
	if err != nil {
		return err
	}
	der, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		return fmt.Errorf("encoding key: %w", err)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, name+".*.tmp") // created with mode 0600
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed
	if err := pem.Encode(tmp, &pem.Block{Type: "PRIVATE KEY", Bytes: der}); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// Promote makes the pending key the current one, replacing it atomically.
func (s Store) Promote(email string) error {
	from, err := s.Path(email, Pending)
	if err != nil {
		return err
	}
	to, err := s.Path(email, Current)
	if err != nil {
		return err
	}
	return os.Rename(from, to)
}
