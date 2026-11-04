package wallets_test

import (
	"delsignrepl/state"
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"delsignrepl/wallets"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	a1 = uitest.Address(1)
	a2 = uitest.Address(2)
	a3 = uitest.Address(3)
)

// setup: wallet 1 has a1 and a2, wallet 2 has a3, wallet 5 has no address yet.
func setup(t *testing.T) (*uitest.Backend, *ui.Shell, *uitest.Sim) {
	t.Helper()
	b := uitest.StartBackend(t, map[int][]string{1: {a1, a2}, 2: {a3}, 5: nil}, 1, 2, 5)
	b.Balances[a2] = "1500000000000000000"
	clear := func() { state.Login("", ""); state.SetContext(0, "") }
	clear()
	t.Cleanup(clear)
	state.Login("a@example.com", "tok")

	app := tview.NewApplication()
	sh := ui.NewShell(app)
	sh.Token = func() string { return state.Token }
	sh.SetHome(tview.NewTextView().SetText("home"))
	return b, sh, uitest.Start(t, app, 120, 30, sh.Run)
}

func TestWalletsTableListsAddressesAndEmptyWallets(t *testing.T) {
	_, sh, sim := setup(t)
	sim.Do(func() { wallets.ShowWallets(sh) })
	if sh.Current() != wallets.WalletsPage {
		t.Fatalf("on %q", sh.Current())
	}
	sim.Shows("Wallet", "Address", a1, a2, a3, "(no address yet)", "n new wallet")
	text := sim.Text()
	if strings.Index(text, a1) > strings.Index(text, a3) || strings.Index(text, a3) > strings.Index(text, "(no address yet)") {
		t.Error("rows should follow walletctx order, then wallets with no address")
	}
}

func TestWalletsEnterSetsContextAndBalance(t *testing.T) {
	_, sh, sim := setup(t)
	sim.Do(func() { wallets.ShowWallets(sh) })
	sim.Key(tcell.KeyDown) // a1 -> a2
	sim.Key(tcell.KeyEnter)
	if state.WalletId != 1 || state.Address != a2 {
		t.Fatalf("context is %d %s, want 1 %s", state.WalletId, state.Address, a2)
	}
	if state.Balance == nil || state.Balance.String() != "1500000000000000000" {
		t.Errorf("balance not fetched: %v (%s)", state.Balance, state.BalanceErr)
	}
	sim.Shows("Using wallet 1 "+state.ShortAddress(a2), "*      1 "+a2)
	if sh.Current() != wallets.WalletsPage {
		t.Error("Enter should stay on the table")
	}

	// A wallet with no address can't be the context.
	sim.Key(tcell.KeyDown)
	sim.Key(tcell.KeyDown) // a3 -> wallet 5
	sim.Key(tcell.KeyEnter)
	if state.Address != a2 {
		t.Error("a wallet with no address became the context")
	}
	sim.Shows("Wallet 5 has no address yet: press a")
}

func TestWalletsCreateWalletThenAddress(t *testing.T) {
	b, sh, sim := setup(t)
	sim.Do(func() { wallets.ShowWallets(sh) })
	sim.Type("n")
	sim.Shows("Created wallet 101. Press a")
	sim.Type("a") // the new wallet is selected, so its address goes there
	got := b.Addresses(101)
	if len(got) != 1 {
		t.Fatalf("wallet 101 has addresses %v", got)
	}
	sim.Shows(got[0], "Created address "+state.ShortAddress(got[0])+" in wallet 101")
	sim.Key(tcell.KeyEnter) // the new address is selected: use it
	if state.Address != got[0] {
		t.Errorf("context %s, want the new address %s", state.Address, got[0])
	}

	// Both are in the activity, the address with itself to copy.
	log := sh.Activity()
	if len(log) != 2 || log[0].What != "Created wallet" || log[0].Copy != "101" ||
		log[1].What != "Created address" || log[1].Copy != got[0] {
		t.Errorf("activity: %+v", log)
	}
}

// c copies the selected row's address; a wallet with no address has nothing to copy.
func TestWalletsCopyAddress(t *testing.T) {
	_, sh, sim := setup(t)
	var copied []string
	prev := ui.SystemCopy
	ui.SystemCopy = func(text string) (string, error) { copied = append(copied, text); return "", nil }
	t.Cleanup(func() { ui.SystemCopy = prev })

	sim.Do(func() { wallets.ShowWallets(sh) })
	sim.Key(tcell.KeyDown) // a2
	sim.Type("c")
	if len(copied) != 1 || copied[0] != a2 || string(sim.Screen.GetClipboardData()) != a2 {
		t.Errorf("copied %q (terminal %q), want %s", copied, sim.Screen.GetClipboardData(), a2)
	}
	sim.Shows("Copied address " + state.ShortAddress(a2))
	sim.Key(tcell.KeyDown)
	sim.Key(tcell.KeyDown) // wallet 5
	sim.Type("c")
	if len(copied) != 1 {
		t.Errorf("copied from a wallet with no address: %q", copied)
	}
	sim.Shows("This wallet has no address to copy yet.")
}

func TestWalletsEmpty(t *testing.T) {
	uitest.StartBackend(t, nil)
	clear := func() { state.Login("", ""); state.SetContext(0, "") }
	clear()
	t.Cleanup(clear)
	app := tview.NewApplication()
	sh := ui.NewShell(app)
	sh.SetHome(tview.NewTextView())
	sim := uitest.Start(t, app, 120, 30, sh.Run)

	sim.Do(func() { wallets.ShowWallets(sh) })
	sim.Shows("No wallets yet: press n")
	sim.Type("a")
	sim.Shows("Create a wallet first (n).")
}

func TestPickAddressExcludesSourceAndGoesBack(t *testing.T) {
	_, sh, sim := setup(t)
	sim.Do(func() { sh.Show("Form", tview.NewTextView().SetText("the form")) })
	var picked string
	sim.Do(func() { wallets.PickAddress(sh, a1, func(a string) { picked = a }) })
	if sh.Current() != wallets.PickPage {
		t.Fatalf("on %q", sh.Current())
	}
	if strings.Contains(sim.Text(), a1) || strings.Contains(sim.Text(), "(no address yet)") {
		t.Error("the picker lists the source or an empty wallet")
	}
	sim.Shows(a2, a3)
	sim.Key(tcell.KeyDown)
	sim.Key(tcell.KeyEnter)
	if picked != a3 || sh.Current() != "Form" {
		t.Errorf("picked %q, on %q", picked, sh.Current())
	}

	picked = ""
	sim.Do(func() { wallets.PickAddress(sh, a1, func(a string) { picked = a }) })
	sim.Key(tcell.KeyEscape)
	if picked != "" || sh.Current() != "Form" {
		t.Errorf("Esc picked %q, on %q", picked, sh.Current())
	}
}
