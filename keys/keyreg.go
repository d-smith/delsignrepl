package keys

import (
	"crypto/rsa"
	"delsignrepl/api"
	"errors"
	"strings"
)

// BuildKeyReg builds a POST /api/v1/keyreg body registering next for email.
// sig4reg is next's signature over email. When current is the registered key being
// replaced, sig4rotate is current's signature over email + next's public key hex.
// current is nil for a first registration, and then sig4rotate is left out of the JSON.
func BuildKeyReg(email string, next, current *rsa.PrivateKey) (api.KeyReg, error) {
	if next == nil {
		return api.KeyReg{}, errors.New("no key to register")
	}
	pub, err := PublicKeyHex(&next.PublicKey)
	if err != nil {
		return api.KeyReg{}, err
	}
	sig4reg, err := Sign(email, next)
	if err != nil {
		return api.KeyReg{}, err
	}
	reg := api.KeyReg{Email: email, PubKey: pub, SignatureForRegistration: sig4reg}
	if current != nil {
		if reg.Sig4Rotate, err = Sign(email+pub, current); err != nil {
			return api.KeyReg{}, err
		}
	}
	return reg, nil
}

// Errors from Register that need the user to act.
var (
	ErrNoKey       = errors.New("there's no key on this machine for this email. Generate one first")
	ErrKeyMismatch = errors.New("this machine's key isn't the one registered with the backend, so sends will be rejected. " +
		"There's no way to replace it without the registered key: ask the backend owner to reset the account")
)

// Outcome is what Register did.
type Outcome int

const (
	Registered        Outcome = iota // first registration of the current key
	Rotated                          // the pending key replaced the registered one and is now current
	Promoted                         // an earlier rotation had succeeded; the pending key is now current
	AlreadyRegistered                // nothing to do
)

func (o Outcome) String() string {
	switch o {
	case Registered:
		return "Key registered."
	case Rotated:
		return "Key replaced. The new key is now in use."
	case Promoted:
		return "An earlier key replacement had already gone through. The new key is now in use."
	case AlreadyRegistered:
		return "This key is already registered. Nothing to do."
	}
	return "unknown outcome"
}

// Register brings the backend's registration in line with the keys in the store
// (plan D2):
//
//   - no key registered: register the current key;
//   - current registered and a pending key exists: rotate to the pending key, then promote it;
//   - pending already registered (an earlier rotation's response was lost): promote it;
//   - current registered, no pending: nothing to do;
//   - registered key matches neither: ErrKeyMismatch.
//
// The pending key is already on disk before any request, so no outcome loses a key.
// A 409 (the key changed meanwhile) reloads the registration and tries once more.
func Register(c *api.Client, s Store, email string) (Outcome, error) {
	out, err := registerOnce(c, s, email)
	if api.StatusOf(err) == 409 {
		out, err = registerOnce(c, s, email)
	}
	return out, err
}

func registerOnce(c *api.Client, s Store, email string) (Outcome, error) {
	current, err := s.Load(email, Current)
	if err != nil {
		return 0, err
	}
	if current == nil {
		return 0, ErrNoKey
	}
	pending, err := s.Load(email, Pending)
	if err != nil {
		return 0, err
	}

	user, err := c.GetUser(email)
	if api.StatusOf(err) == 404 {
		reg, err := BuildKeyReg(email, current, nil)
		if err != nil {
			return 0, err
		}
		return Registered, c.RegisterKey(reg)
	}
	if err != nil {
		return 0, err
	}

	if pending != nil {
		pendingHex, err := PublicKeyHex(&pending.PublicKey)
		if err != nil {
			return 0, err
		}
		if strings.EqualFold(user.PubKey, pendingHex) {
			return Promoted, s.Promote(email)
		}
	}
	currentHex, err := PublicKeyHex(&current.PublicKey)
	if err != nil {
		return 0, err
	}
	if !strings.EqualFold(user.PubKey, currentHex) {
		return 0, ErrKeyMismatch
	}
	if pending == nil {
		return AlreadyRegistered, nil
	}

	reg, err := BuildKeyReg(email, pending, current)
	if err != nil {
		return 0, err
	}
	if err := c.RegisterKey(reg); err != nil {
		return 0, err
	}
	return Rotated, s.Promote(email)
}

// Status is what CheckKey found. It changes nothing.
type Status int

const (
	NoLocalKey     Status = iota // no current key on this machine
	NotRegistered                // a local key, but none registered
	Matches                      // the local key is the registered one
	PendingReady                 // matches, and a replacement is waiting to be registered
	FinishRotation               // the pending key is already registered; Register will promote it
	Mismatch                     // the registered key isn't this machine's
)

func (st Status) String() string {
	switch st {
	case NoLocalKey:
		return "No signing key on this machine for this email. Generate one, then register it."
	case NotRegistered:
		return "Your key isn't registered yet. Register it."
	case Matches:
		return "Your key is registered and matches this machine's key."
	case PendingReady:
		return "Your key is registered. A replacement key is waiting: register it to switch to it."
	case FinishRotation:
		return "A key replacement went through but wasn't recorded here. Register again to finish it."
	case Mismatch:
		return ErrKeyMismatch.Error() + "."
	}
	return "unknown status"
}

// CheckKey compares the store's keys with the registered one, read-only.
func CheckKey(c *api.Client, s Store, email string) (Status, error) {
	current, err := s.Load(email, Current)
	if err != nil {
		return 0, err
	}
	if current == nil {
		return NoLocalKey, nil
	}
	pending, err := s.Load(email, Pending)
	if err != nil {
		return 0, err
	}
	user, err := c.GetUser(email)
	if api.StatusOf(err) == 404 {
		return NotRegistered, nil
	}
	if err != nil {
		return 0, err
	}
	if pending != nil {
		pendingHex, err := PublicKeyHex(&pending.PublicKey)
		if err != nil {
			return 0, err
		}
		if strings.EqualFold(user.PubKey, pendingHex) {
			return FinishRotation, nil
		}
	}
	currentHex, err := PublicKeyHex(&current.PublicKey)
	if err != nil {
		return 0, err
	}
	switch {
	case !strings.EqualFold(user.PubKey, currentHex):
		return Mismatch, nil
	case pending != nil:
		return PendingReady, nil
	default:
		return Matches, nil
	}
}
