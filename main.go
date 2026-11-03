package main

import (
	"delsignrepl/keys"
	"delsignrepl/send"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/ui"
	"delsignrepl/wallets"

	"github.com/rivo/tview"
)

var appToken string

// buildShell lays out the app: the existing screens run in the shell's content area and
// the old menu's actions sit in the sidebar (plan/03, Phase 0).
func buildShell(app *tview.Application) *ui.Shell {
	sh := ui.NewShell(app)
	pages := sh.Content

	sh.AddAction("Generate key", 'k', func() { keys.DoKeyGeneration(pages) }).
		AddAction("Register key", 'r', func() { keys.DoKeyRegistration(pages, appToken) }).
		AddAction("Key status", 'i', func() { keys.DoKeyStatus(pages, appToken) }).
		AddAction("Create wallet", 'w', func() { wallets.DoWalletGeneration(pages, appToken) }).
		AddAction("Generate address", 'a', func() { wallets.DoAddressGeneration(pages, appToken) }).
		AddAction("Set wallet context", 'c', func() { state.DoSetWalletAndAccountCtx(pages, appToken) }).
		AddAction("Get balance", 'b', func() {
			wallets.DoGetBalance(pages, state.WalletId, state.Address, appToken)
		}).
		AddAction("Send ETH", 's', func() { send.ShowSendForm(pages, appToken) }).
		AddAction("Log in again", 't', func() { pages.SwitchToPage(ui.LoginPage) }).
		AddAction("Quit", 'q', app.Stop)

	home := tview.NewTextView().SetText("\n  Choose an action from the menu on the left.")
	pages.AddPage(ui.HomePage, home, true, true)

	onLogin := func(email string) { keys.OnLogin(pages, email, appToken) }
	pages.AddPage(ui.LoginPage, token.GetTokenInputForm(pages, app, &appToken, onLogin), true, true)
	return sh
}

func main() {
	if err := buildShell(tview.NewApplication()).Run(); err != nil {
		panic(err)
	}
}
