package main

import (
	"delsignrepl/state"
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	addr1 = "0x" + strings.Repeat("1a", 20)
	addr2 = "0x" + strings.Repeat("2b", 20)
)

// fakeBackend answers the wallet calls the screens make, from openapi.yaml's shapes.
func fakeBackend(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/v1/wallets":
			json.NewEncoder(w).Encode([]int{1, 2})
		case "POST /api/v1/wallets":
			w.WriteHeader(201)
			json.NewEncoder(w).Encode(map[string]int{"id": 3})
		case "GET /api/v1/walletctx":
			json.NewEncoder(w).Encode([]map[string]any{
				{"walletId": 1, "address": addr1}, {"walletId": 2, "address": addr2}})
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", 404)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DELSIGN_API_URL", srv.URL)
	t.Setenv("DELSIGN_DEV_JWT_SECRET", "")
}

func startApp(t *testing.T) (*ui.Shell, *uitest.Sim) {
	t.Helper()
	app := tview.NewApplication()
	sh := buildShell(app)
	return sh, uitest.Start(t, app, 120, 30, sh.Run)
}

// The app starts on the login page in the shell; Esc goes Home; an action's modal opens
// over Home and OK returns there with the sidebar focused.
func TestAppStartsOnLoginInsideShell(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Shows("Log in with a JWT", "Not logged in", "Generate key", "Send ETH", "Quit")
	if sh.Current() != ui.LoginPage || sim.App.GetFocus() == sh.Sidebar {
		t.Fatal("the login page should be open and focused at start")
	}

	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
		t.Fatalf("Esc from login went to %q", sh.Current())
	}

	sim.Type("i") // Key status, before logging in
	sim.Shows("Log in first")
	sim.Key(tcell.KeyEnter)
	sim.Shows("Choose an action")
	if sim.App.GetFocus() != sh.Sidebar {
		t.Error("OK on the modal should return to Home and the sidebar")
	}

	sim.Type("w") // Create wallet: a modal now, not a page waiting for 'm'
	sim.Shows("Created wallet 3")
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
		t.Errorf("Esc on the wallet modal: on %q", sh.Current())
	}
}

// Every screen reachable from the sidebar backs out to Home with Esc.
func TestEscFromEveryScreen(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape) // leave login
	sim.Do(func() { state.WalletId, state.Address = 1, addr1 })
	t.Cleanup(func() { state.WalletId, state.Address = 0, "" })

	for _, c := range []struct{ key, page, shows string }{
		{"a", "Address Gen", "Wallet ID:"},
		{"c", "SetCtx", "Wallet | address:"},
		{"s", "SendEth", "Destination address:"},
		{"?", ui.HelpPage, "Esc               back"},
		{"t", ui.LoginPage, "Log in with a JWT"},
	} {
		sim.Type(c.key)
		sim.Shows(c.shows)
		if sh.Current() != c.page {
			t.Errorf("%q opened %q, want %q", c.key, sh.Current(), c.page)
		}
		sim.Key(tcell.KeyEscape)
		if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
			t.Errorf("Esc from %q went to %q", c.page, sh.Current())
		}
	}
}

// Esc closes an open dropdown first, and only the next Esc leaves the screen.
func TestEscClosesDropdownFirst(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape)
	sim.Type("c")
	sim.Key(tcell.KeyEnter) // open the wallet/address dropdown
	sim.Shows(addr2)
	sim.Key(tcell.KeyEscape)
	if sh.Current() != "SetCtx" {
		t.Fatalf("Esc with the dropdown open left the screen, on %q", sh.Current())
	}
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage {
		t.Errorf("second Esc went to %q", sh.Current())
	}
}

// The send form's picker returns to the form with the address filled in, and Cancel on
// the form goes back Home.
func TestSendPickerReturnsToForm(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape)
	sim.Do(func() { state.WalletId, state.Address = 1, addr1 })
	t.Cleanup(func() { state.WalletId, state.Address = 0, "" })

	sim.Type("s")
	sim.Key(tcell.KeyTab) // destination -> "Use Wallet Address"
	sim.Type(" ")         // tick it: opens the picker
	if sh.Current() != "WalletAddress" {
		t.Fatalf("picker didn't open, on %q", sh.Current())
	}
	sim.Shows("Send to one of my addresses", addr2)
	sim.Key(tcell.KeyTab) // dropdown -> "Use this"
	sim.Key(tcell.KeyEnter)
	if sh.Current() != "SendEth" {
		t.Fatalf("Use this went to %q, want the send form", sh.Current())
	}
	sim.Shows(addr2)

	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage {
		t.Errorf("Esc from the send form went to %q", sh.Current())
	}
}
