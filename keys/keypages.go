package keys

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"errors"
	"fmt"

	"github.com/rivo/tview"
)

const lostKeyWarning = "This file is your only way to authorise sends. If it's lost, funds in your " +
	"addresses are locked unless the backend owner resets your account."

func showMessage(pages *tview.Pages, name, text string) {
	modal := tview.NewModal().
		SetText(text).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.SwitchToPage("Menu")
		})
	pages.AddPage(name, modal, true, false)
	pages.SwitchToPage(name)
}

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
func OnLogin(pages *tview.Pages, email, appToken string) {
	state.Email = email
	s := StoreFromEnv()
	if err := loadCurrent(s); err != nil {
		ui.ShowError(pages, "Error loading your key", err, appToken, "Menu")
		return
	}
	DoKeyStatus(pages, appToken)
}

// DoKeyStatus shows whether this machine's key is the registered one. It changes nothing.
func DoKeyStatus(pages *tview.Pages, appToken string) {
	if state.Email == "" {
		showMessage(pages, "KeyStatus", "Log in first (t).")
		return
	}
	st, err := CheckKey(api.New(appToken), StoreFromEnv(), state.Email)
	if err != nil {
		ui.ShowError(pages, "Error checking your key", err, appToken, "Menu")
		return
	}
	showMessage(pages, "KeyStatus", fmt.Sprintf("Signed in as %s.\n\n%s", state.Email, st))
}

// DoKeyGeneration makes the current key if there's none, or else (after confirmation) a
// pending replacement. It never overwrites the current key.
func DoKeyGeneration(pages *tview.Pages) {
	if state.Email == "" {
		showMessage(pages, "Keygen", "Log in first (t).")
		return
	}
	s := StoreFromEnv()
	fail := func(err error) {
		showMessage(pages, "Keygen", "Error generating key: "+err.Error())
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
		showMessage(pages, "Keygen", fmt.Sprintf("Signing key saved to %s.\n\n%s\n\nNow register it (r).", path, lostKeyWarning))
		return
	}

	pending, err := s.Load(state.Email, Pending)
	if err != nil {
		fail(err)
		return
	}
	if pending != nil {
		showMessage(pages, "Keygen", "A replacement key is already waiting. Register it (r) before making another.")
		return
	}

	confirm := tview.NewModal().
		SetText("You already have a key. Make a replacement?\n\n" +
			"Your current key stays in use until the replacement is registered (r).").
		AddButtons([]string{"Replace", "Cancel"}).
		SetDoneFunc(func(_ int, label string) {
			if label != "Replace" {
				pages.SwitchToPage("Menu")
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
			showMessage(pages, "Keygen", fmt.Sprintf("Replacement key saved to %s.\n\n%s\n\nRegister it (r) to switch to it.", path, lostKeyWarning))
		})
	pages.AddPage("KeygenConfirm", confirm, true, false)
	pages.SwitchToPage("KeygenConfirm")
}

// DoKeyRegistration registers the key, or rotates to a pending replacement (plan D2).
func DoKeyRegistration(pages *tview.Pages, appToken string) {
	if state.Email == "" {
		showMessage(pages, "KeyReg", "Log in first (t).")
		return
	}
	s := StoreFromEnv()
	out, err := Register(api.New(appToken), s, state.Email)
	if errors.Is(err, ErrNoKey) || errors.Is(err, ErrKeyMismatch) {
		showMessage(pages, "KeyReg", "Can't register: "+err.Error()+".")
		return
	}
	if err != nil {
		ui.ShowError(pages, "Error registering key", err, appToken, "Menu")
		return
	}
	if err := loadCurrent(s); err != nil {
		ui.ShowError(pages, "Key registered, but loading it failed", err, appToken, "Menu")
		return
	}
	showMessage(pages, "KeyReg", out.String())
}
