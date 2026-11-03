package keys

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"errors"
	"fmt"
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
	state.Email = email
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
	st, err := CheckKey(api.New(sh.Token()), StoreFromEnv(), state.Email)
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
	sh.Message(out.String())
}
