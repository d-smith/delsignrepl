package main

import (
	"delsignrepl/keys"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const testEmail = "a@example.com"

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
		case "GET /api/v1/authping":
		case "GET /api/v1/users/" + testEmail:
			http.Error(w, "No key registered", 404)
		case "GET /api/v1/wallets/balance/" + addr1:
			json.NewEncoder(w).Encode(map[string]string{"address": addr1, "amount": "49000000000000000"})
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
	t.Setenv("DELSIGN_KEY_DIR", t.TempDir())
	resetSession(t)
}

// resetSession clears the session globals before and after a test.
func resetSession(t *testing.T) {
	t.Helper()
	clear := func() {
		state.Login("", "")
		state.SetContext(0, "")
	}
	clear()
	t.Cleanup(clear)
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
	sim.Shows("Next: log in (t).")
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
	sim.Do(func() { state.SetContext(1, addr1) })

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
	sim.Do(func() { state.SetContext(1, addr1) })

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

// Logging in through the form, refreshing, and setting the context fill the header and
// Home from the session, with no modal needed to see where you are.
func TestHeaderAndHomeShowTheSession(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Shows("Not logged in")

	tok, err := token.GenerateToken(time.Hour, &token.TokenFields{Id: "x", Email: testEmail}, "any")
	if err != nil {
		t.Fatal(err)
	}
	sim.Type(tok)
	sim.Key(tcell.KeyTab)   // Save
	sim.Key(tcell.KeyEnter) // log in: key status modal over Home
	sim.Key(tcell.KeyEnter) // OK
	if sh.Current() != ui.HomePage {
		t.Fatalf("after login, on %q", sh.Current())
	}
	sim.Shows(testEmail+"  |  key: none on this machine  |  no wallet context",
		"Next: generate a signing key (k)")

	// A key made outside the app: Refresh (R) picks it up and checks it.
	k, err := keys.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := keys.StoreFromEnv().Save(testEmail, keys.Current, k); err != nil {
		t.Fatal(err)
	}
	sim.Type("R")
	sim.Shows("key: not registered", "Next: register your key (r).", "Refreshed at")

	sim.Type("c")
	sim.Key(tcell.KeyTab)   // Use this
	sim.Key(tcell.KeyEnter) // context set, balance fetched, back Home
	if sh.Current() != ui.HomePage {
		t.Fatalf("after setting the context, on %q", sh.Current())
	}
	sim.Shows("wallet 1 0x1a1a...1a1a  |  0.049 ETH",
		"Address    "+addr1, "0.049 ETH (49000000000000000 wei)")
}
