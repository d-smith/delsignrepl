package main

import (
	"delsignrepl/keys"
	"delsignrepl/send"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"delsignrepl/wallets"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const testEmail = "a@example.com"

var (
	addr1 = uitest.Address(1)
	addr2 = uitest.Address(2)
)

// fakeBackend: wallet 1 holds addr1 (0.049 ETH), wallet 2 holds addr2.
func fakeBackend(t *testing.T) *uitest.Backend {
	t.Helper()
	b := uitest.StartBackend(t, map[int][]string{1: {addr1}, 2: {addr2}}, 1, 2)
	b.Balances[addr1] = "49000000000000000"
	t.Setenv("DELSIGN_DEV_JWT_SECRET", "")
	t.Setenv("DELSIGN_KEY_DIR", t.TempDir())
	resetSession(t)
	return b
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
	sim.Shows("Log in with a JWT", "Not logged in", "Keys", "Send ETH", "Quit")
	if sh.Current() != ui.LoginPage || sim.App.GetFocus() == sh.Sidebar {
		t.Fatal("the login page should be open and focused at start")
	}
	if main, _ := sh.Sidebar.GetItemText(sh.Sidebar.GetCurrentItem()); main != "Log in again" {
		t.Errorf("at start the menu highlights %q, want the login item", main)
	}

	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
		t.Fatalf("Esc from login went to %q", sh.Current())
	}

	sim.Type("k") // Keys, before logging in
	sim.Shows("Log in first")
	sim.Key(tcell.KeyEnter)
	sim.Shows("Next: log in (l).")
	if sim.App.GetFocus() != sh.Sidebar {
		t.Error("OK on the modal should return to Home and the sidebar")
	}

	sim.Type("s") // Send with no address chosen: a modal over Home
	sim.Shows("No address chosen yet")
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
		t.Errorf("Esc on the modal: on %q", sh.Current())
	}
}

// Every screen reachable from the sidebar backs out to Home with Esc.
func TestEscFromEveryScreen(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape) // leave login
	sim.Do(func() {
		state.Login(testEmail, "tok")
		state.SetContext(1, addr1)
	})

	for _, c := range []struct{ key, page, shows string }{
		{"k", keys.KeysPage, "Replacement"},
		{"w", wallets.WalletsPage, "n new wallet"},
		{"s", send.SendPage, "Amount (wei):"},
		{"?", ui.HelpPage, "Esc               back"},
		{"l", ui.LoginPage, "Log in with a JWT"},
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

// The send form: the ETH preview follows the amount as it's typed, and "Pick from my
// addresses" returns to the form with the destination filled in and the amount kept.
func TestSendFormPreviewAndPicker(t *testing.T) {
	fakeBackend(t)
	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape)
	sim.Do(func() { state.SetContext(1, addr1) })

	sim.Type("s")
	sim.Shows("From:", addr1, "To:", "Amount (wei):")
	sim.Key(tcell.KeyTab) // To -> Amount
	sim.Type("1000000000000000")
	sim.Shows("= 0.001 ETH")
	sim.Type("0")
	sim.Shows("= 0.01 ETH")

	sim.Key(tcell.KeyTab) // Amount -> Send (the preview is skipped)
	sim.Key(tcell.KeyTab) // Send -> Pick from my addresses
	sim.Key(tcell.KeyEnter)
	if sh.Current() != wallets.PickPage {
		t.Fatalf("picker didn't open, on %q", sh.Current())
	}
	sim.Shows("Send to one of my addresses", addr2)
	sim.Key(tcell.KeyEnter) // addr2, the only other address
	if sh.Current() != send.SendPage {
		t.Fatalf("picking went to %q, want the send form", sh.Current())
	}
	sim.Shows(addr2, "10000000000000000", "= 0.01 ETH")

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
	sim.Key(tcell.KeyEnter) // log in: Home, no modal
	if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
		t.Fatalf("after login, on %q", sh.Current())
	}
	if main, _ := sh.Sidebar.GetItemText(sh.Sidebar.GetCurrentItem()); main != "Keys" {
		t.Errorf("after login the menu highlights %q, want Keys", main)
	}
	sim.Shows(testEmail+"  |  key: none on this machine  |  no wallet context",
		"Next: open Keys (k), generate a signing key (g)", "Logged in as "+testEmail)

	// A key made outside the app: Refresh (R) picks it up and checks it.
	k, err := keys.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := keys.StoreFromEnv().Save(testEmail, keys.Current, k); err != nil {
		t.Fatal(err)
	}
	sim.Type("R")
	sim.Shows("key: not registered", "Next: open Keys (k) and register your key (r).", "Refreshed at")

	sim.Type("w")
	sim.Key(tcell.KeyEnter) // use addr1, the first row: context set, balance fetched
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage {
		t.Fatalf("after setting the context, on %q", sh.Current())
	}
	sim.Shows("wallet 1 "+state.ShortAddress(addr1)+"  |  0.049 ETH",
		"Address    "+addr1, "0.049 ETH (49000000000000000 wei)")
}

// The Keys screen: generate (g), register (r), make a replacement and rotate to it, with
// the status and files on screen after each step.
func TestKeysScreen(t *testing.T) {
	b := fakeBackend(t)
	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape)
	sim.Do(func() { state.Login(testEmail, "tok") })
	store := keys.StoreFromEnv()
	current, _ := store.Path(testEmail, keys.Current)
	pending, _ := store.Path(testEmail, keys.Pending)

	sim.Type("k")
	if sh.Current() != keys.KeysPage {
		t.Fatalf("k opened %q", sh.Current())
	}
	sim.Shows("Status       none on this machine", "Key file     none", "Replacement  none",
		"Next: open Keys (k), generate")

	sim.Type("g")
	sim.Shows("Signing key saved to", "Now register it (r).")
	sim.Key(tcell.KeyEnter)
	sim.Shows("Status       not registered", "Key file     "+current)
	if sh.Current() != keys.KeysPage {
		t.Fatalf("OK after generating went to %q", sh.Current())
	}

	sim.Type("r")
	sim.Shows("Key registered.")
	sim.Key(tcell.KeyEnter)
	sim.Shows("Status       matches", "registered and matches this machine's key")
	first := b.Registered(testEmail)
	if first == "" {
		t.Fatal("nothing registered")
	}

	sim.Type("g") // a key exists: offer a replacement
	sim.Shows("You already have a key. Make a", "Replace     Cancel")
	sim.Key(tcell.KeyEnter) // Replace
	sim.Shows("Replacement key saved to")
	sim.Key(tcell.KeyEnter)
	sim.Shows("matches, replacement waiting", "Replacement  "+pending)

	sim.Type("r") // rotate
	sim.Shows("Key replaced.")
	sim.Key(tcell.KeyEnter)
	sim.Shows("Status       matches (", "Replacement  none")
	if b.Registered(testEmail) == first {
		t.Error("rotation didn't change the registered key")
	}

	var whats []string
	for _, e := range sh.Activity() {
		whats = append(whats, e.What)
	}
	if got := strings.Join(whats, ", "); got != "Generated key, Registered key, Generated replacement, Replaced key" {
		t.Errorf("activity: %s", got)
	}

	sim.Type("R")
	sim.Shows("Key checked at")
	sim.Key(tcell.KeyEscape)
	sim.Shows(testEmail + "  |  key: matches")
}

// With the dev secret set, the login page has an Email field and the cheat code button,
// which makes a token for that email. Editing the email takes the token back.
func TestCheatCodeLogin(t *testing.T) {
	fakeBackend(t)
	t.Setenv("DELSIGN_DEV_JWT_SECRET", "s3cret")
	sh, sim := startApp(t)
	sim.Shows("Email:", "Token:", token.CheatCodeLabel)

	sim.Key(tcell.KeyTab)   // Email -> Token
	sim.Key(tcell.KeyTab)   // Token -> Apply Cheat Code
	sim.Key(tcell.KeyEnter) // no email yet
	sim.Shows("Enter an email address in the Email")
	sim.Key(tcell.KeyEnter)

	sim.Key(tcell.KeyBacktab)
	sim.Key(tcell.KeyBacktab) // back to Email
	sim.Type(testEmail)
	sim.Key(tcell.KeyTab)
	sim.Key(tcell.KeyTab)
	sim.Key(tcell.KeyEnter)
	sim.Shows("Token made for "+testEmail, "Token:  eyJhbGci")

	sim.Key(tcell.KeyBacktab)
	sim.Key(tcell.KeyBacktab)
	sim.Type("x") // a different email: the token goes
	if strings.Contains(sim.Text(), "eyJ") {
		t.Fatal("editing the email kept the token minted for the old one")
	}
	sim.Key(tcell.KeyBackspace2)
	sim.Key(tcell.KeyTab)
	sim.Key(tcell.KeyTab)
	sim.Key(tcell.KeyEnter) // make it again
	sim.Key(tcell.KeyTab)   // Save
	sim.Key(tcell.KeyEnter)
	if sh.Current() != ui.HomePage {
		t.Fatalf("after Save, on %q", sh.Current())
	}
	sim.Shows(testEmail + "  |  key: none on this machine")
}

// A send through the form is logged in Activity with its txnid, which c copies.
func TestSendIsLoggedInActivity(t *testing.T) {
	fakeBackend(t)
	var copied []string
	prev := ui.SystemCopy
	ui.SystemCopy = func(text string) (string, error) { copied = append(copied, text); return "", nil }
	t.Cleanup(func() { ui.SystemCopy = prev })

	sh, sim := startApp(t)
	sim.Key(tcell.KeyEscape)
	k, err := keys.Generate()
	if err != nil {
		t.Fatal(err)
	}
	sim.Do(func() {
		state.Login(testEmail, "tok")
		state.PrivateKey = k
		state.SetContext(1, addr1)
	})

	sim.Type("s")
	sim.Type(addr2)
	sim.Key(tcell.KeyTab)
	sim.Type("1000000000000000")
	sim.Key(tcell.KeyTab)   // Send
	sim.Key(tcell.KeyEnter) // confirmation
	sim.Shows("Send 1000000000000000 wei (0.001 ETH)")
	sim.Key(tcell.KeyEnter) // Confirm
	sim.Shows("Broadcast. Transaction", "Copy txnid")
	sim.Key(tcell.KeyTab)   // OK -> Copy txnid
	sim.Key(tcell.KeyEnter) // copies, then Home
	if sh.Current() != ui.HomePage || len(copied) != 1 || copied[0] != uitest.Txnid(1) {
		t.Fatalf("Copy txnid: on %q, copied %q", sh.Current(), copied)
	}
	sim.Shows("Copied txnid through the terminal.")

	sim.Type("a")
	if sh.Current() != ui.ActivityPage {
		t.Fatalf("a opened %q", sh.Current())
	}
	sim.Shows("Send: sent", "0.001 ETH to "+state.ShortAddress(addr2)+"  tx "+uitest.Txnid(1)[:20])
	sim.Type("c")
	if len(copied) != 2 || copied[1] != uitest.Txnid(1) {
		t.Errorf("copied %q, want the txnid again", copied)
	}
	if string(sim.Screen.GetClipboardData()) != uitest.Txnid(1) {
		t.Errorf("terminal clipboard %q", sim.Screen.GetClipboardData())
	}
	sim.Key(tcell.KeyEnter) // details, one value per line
	sim.Shows("From       "+addr1, "To         "+addr2, "(1000000000000000 wei)", "Request id ", "Txnid      "+uitest.Txnid(1))
}
