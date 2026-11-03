package keys

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"errors"
	"fmt"
	"time"
)

const lostKeyWarning = "This file is your only way to authorise sends. If it's lost, funds in your " +
	"addresses are locked unless the backend owner resets your account."

// loadCurrent makes the stored current key for the logged-in email the signing key.
func loadCurrent(s Store) error {
	k, err := s.Load(state.Email, Current)
	if err != nil {
		state.PrivateKey = nil
		return err
	}
	state.PrivateKey = k
	return nil
}

// OnLogin runs after a token is accepted: it loads this email's key and shows whether it
// matches the registered one.
func OnLogin(sh *ui.Shell, email string) {
	state.Login(email, sh.Token())
	sh.Home()
	if err := loadCurrent(StoreFromEnv()); err != nil {
		sh.Error("Error loading your key", err)
		return
	}
	DoKeyStatus(sh)
}

// DoKeyStatus shows whether this machine's key is the registered one. It changes nothing.
func DoKeyStatus(sh *ui.Shell) {
	if state.Email == "" {
		sh.Message("Log in first (t).")
		return
	}
	st, err := RefreshKeyStatus(sh.Token())
	if err != nil {
		sh.Error("Error checking your key", err)
		return
	}
	sh.Message(fmt.Sprintf("Signed in as %s.\n\n%s", state.Email, st))
}

// DoKeyGeneration makes the current key if there's none, or else (after confirmation) a
// pending replacement. It never overwrites the current key.
func DoKeyGeneration(sh *ui.Shell) {
	if state.Email == "" {
		sh.Message("Log in first (t).")
		return
	}
	s := StoreFromEnv()
	fail := func(err error) {
		sh.Message("Error generating key: " + err.Error())
	}

	current, err := s.Load(state.Email, Current)
	if err != nil {
		fail(err)
		return
	}
	if current == nil {
		k, err := Generate()
		if err == nil {
			err = s.Save(state.Email, Current, k)
		}
		if err != nil {
			fail(err)
			return
		}
		state.PrivateKey = k
		refreshQuietly(sh)
		path, _ := s.Path(state.Email, Current)
		sh.Message(fmt.Sprintf("Signing key saved to %s.\n\n%s\n\nNow register it (r).", path, lostKeyWarning))
		return
	}

	pending, err := s.Load(state.Email, Pending)
	if err != nil {
		fail(err)
		return
	}
	if pending != nil {
		sh.Message("A replacement key is already waiting. Register it (r) before making another.")
		return
	}

	sh.Modal("You already have a key. Make a replacement?\n\n"+
		"Your current key stays in use until the replacement is registered (r).",
		[]string{"Replace", "Cancel"}, func(label string) {
			if label != "Replace" {
				return
			}
			k, err := Generate()
			if err == nil {
				err = s.Save(state.Email, Pending, k)
			}
			if err != nil {
				fail(err)
				return
			}
			refreshQuietly(sh)
			path, _ := s.Path(state.Email, Pending)
			sh.Message(fmt.Sprintf("Replacement key saved to %s.\n\n%s\n\nRegister it (r) to switch to it.", path, lostKeyWarning))
		})
}

// DoKeyRegistration registers the key, or rotates to a pending replacement (plan D2).
func DoKeyRegistration(sh *ui.Shell) {
	if state.Email == "" {
		sh.Message("Log in first (t).")
		return
	}
	s := StoreFromEnv()
	out, err := Register(api.New(sh.Token()), s, state.Email)
	if errors.Is(err, ErrNoKey) || errors.Is(err, ErrKeyMismatch) {
		sh.Message("Can't register: " + err.Error() + ".")
		return
	}
	if err != nil {
		sh.Error("Error registering key", err)
		return
	}
	if err := loadCurrent(s); err != nil {
		sh.Error("Key registered, but loading it failed", err)
		return
	}
	refreshQuietly(sh)
	sh.Message(out.String())
}

// Label is the status's short form, for the header.
func (st Status) Label() string {
	switch st {
	case NoLocalKey:
		return "none on this machine"
	case NotRegistered:
		return "not registered"
	case Matches:
		return "matches"
	case PendingReady:
		return "matches, replacement waiting"
	case FinishRotation:
		return "replacement registered, finish it"
	case Mismatch:
		return "MISMATCH"
	}
	return "unknown"
}

// Next is the step the status calls for, or "" if none.
func (st Status) Next() string {
	switch st {
	case NoLocalKey:
		return "generate a signing key (k), then register it (r)."
	case NotRegistered:
		return "register your key (r)."
	case PendingReady:
		return "register the replacement key (r) to switch to it."
	case FinishRotation:
		return "register (r) to finish switching to the replacement key."
	case Mismatch:
		return "this machine's key isn't the registered one, so sends will be rejected. " +
			"Use the right key file, or ask the backend owner to reset your account."
	}
	return ""
}

// RefreshKeyStatus checks the key (read-only) and records the result in state for the
// header and Home.
func RefreshKeyStatus(token string) (Status, error) {
	st, err := CheckKey(api.New(token), StoreFromEnv(), state.Email)
	if err != nil {
		state.KeyStatus, state.KeyNext = "couldn't check", ""
		return st, err
	}
	state.KeyStatus, state.KeyNext, state.KeyCheckedAt = st.Label(), st.Next(), time.Now()
	return st, nil
}

// refreshQuietly updates the key status after a local change. A failure only leaves the
// header saying "couldn't check"; the action's own message still shows.
func refreshQuietly(sh *ui.Shell) {
	_, _ = RefreshKeyStatus(sh.Token())
}
