package keys

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// KeysPage is the Keys screen's page name.
const KeysPage = "Keys"

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

// OnLogin runs after a token is accepted: it loads this email's key, checks it, and goes
// Home, where the header and Home show the result and the next step.
func OnLogin(sh *ui.Shell, email string) {
	state.Login(email, sh.Token())
	sh.Home()
	if err := loadCurrent(StoreFromEnv()); err != nil {
		sh.Error("Error loading your key", err)
		return
	}
	if _, err := RefreshKeyStatus(sh.Token()); err != nil {
		sh.Error("Error checking your key", err)
		return
	}
	sh.Status("Logged in as " + email)
}

// ShowKeys opens the Keys screen: the key's status and files, with buttons (and keys g,
// r and R) to generate, register and check again. It checks the key on opening.
func ShowKeys(sh *ui.Shell) {
	if state.Email == "" {
		sh.Message("Log in first (t).")
		return
	}
	info := ui.NewLiveText(keysText)
	form := tview.NewForm().
		AddButton("Generate (g)", func() { DoKeyGeneration(sh) }).
		AddButton("Register (r)", func() { DoKeyRegistration(sh) }).
		AddButton("Check again (R)", func() { checkAgain(sh) })
	form.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() != tcell.KeyRune {
			return ev
		}
		switch ev.Rune() {
		case 'g':
			DoKeyGeneration(sh)
		case 'r':
			DoKeyRegistration(sh)
		case 'R':
			checkAgain(sh)
		default:
			return ev
		}
		return nil
	})
	page := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(info, 0, 1, false).
		AddItem(form, 3, 0, true)
	page.SetBorder(true).SetTitle(" Keys ").SetTitleAlign(tview.AlignLeft)
	sh.Show(KeysPage, page)
	if _, err := RefreshKeyStatus(sh.Token()); err != nil {
		sh.Error("Error checking your key", err)
	}
}

func checkAgain(sh *ui.Shell) {
	if _, err := RefreshKeyStatus(sh.Token()); err != nil {
		sh.Error("Error checking your key", err)
		return
	}
	sh.Status("Key checked at " + state.KeyCheckedAt.Format("15:04:05"))
}

// keysText is the Keys screen's text, from the session and the key files on disk.
func keysText() string {
	var b strings.Builder
	line := func(label, value string) { fmt.Fprintf(&b, "  %-12s %s\n", label, value) }
	b.WriteString("\n")
	line("Signed in", state.Email)
	status := "-"
	if state.KeyStatus != "" {
		status = state.KeyStatus
	}
	if !state.KeyCheckedAt.IsZero() {
		status += " (checked " + state.KeyCheckedAt.Format("15:04:05") + ")"
	}
	line("Status", status)
	if state.KeyDetail != "" {
		line("", state.KeyDetail)
	}
	s := StoreFromEnv()
	line("Key file", keyFile(s, Current))
	line("Replacement", keyFile(s, Pending))
	if state.KeyNext != "" {
		b.WriteString("\n  Next: " + state.KeyNext + "\n")
	}
	b.WriteString("\n  g generates your key, or a replacement if you have one; the key in use is\n" +
		"  never overwritten. r registers it, or switches to the replacement. R checks again.\n")
	return b.String()
}

// keyFile is the named key file's path, or "none" if it doesn't exist.
func keyFile(s Store, name string) string {
	path, err := s.Path(state.Email, name)
	if err != nil {
		return "-"
	}
	if _, err := os.Stat(path); err != nil {
		return "none"
	}
	return path
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
		logKey(sh, "Generated key", path)
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
			logKey(sh, "Generated replacement", path)
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
	if out != AlreadyRegistered {
		path, _ := s.Path(state.Email, Current)
		logKey(sh, map[Outcome]string{Registered: "Registered key", Rotated: "Replaced key",
			Promoted: "Finished key replacement"}[out], path)
	}
	sh.Message(out.String())
}

// logKey records a key change in the activity, with the key file's path to copy.
func logKey(sh *ui.Shell, what, path string) {
	sh.Log(ui.Entry{What: what, Summary: state.Email + "  " + path,
		Details: []string{"Email " + state.Email, "File  " + path}, Copy: path, CopyIs: "key file path"})
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

// Next is the step the status calls for, or "" if none. Home shows it, so it names the
// Keys screen (k) as well as the key to press there.
func (st Status) Next() string {
	switch st {
	case NoLocalKey:
		return "open Keys (k), generate a signing key (g), then register it (r)."
	case NotRegistered:
		return "open Keys (k) and register your key (r)."
	case PendingReady:
		return "open Keys (k) and register the replacement key (r) to switch to it."
	case FinishRotation:
		return "open Keys (k) and register (r) to finish switching to the replacement key."
	case Mismatch:
		return "this machine's key isn't the registered one, so sends will be rejected. " +
			"Use the right key file, or ask the backend owner to reset your account."
	}
	return ""
}

// RefreshKeyStatus checks the key (read-only) and records the result in state for the
// header, Home and the Keys screen.
func RefreshKeyStatus(token string) (Status, error) {
	st, err := CheckKey(api.New(token), StoreFromEnv(), state.Email)
	if err != nil {
		state.KeyStatus, state.KeyDetail, state.KeyNext = "couldn't check", "", ""
		return st, err
	}
	state.KeyStatus, state.KeyDetail, state.KeyNext = st.Label(), st.String(), st.Next()
	state.KeyCheckedAt = time.Now()
	return st, nil
}

// refreshQuietly updates the key status after a local change. A failure only leaves the
// header saying "couldn't check"; the action's own message still shows.
func refreshQuietly(sh *ui.Shell) {
	_, _ = RefreshKeyStatus(sh.Token())
}
