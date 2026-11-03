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

// buildShell lays out the app: the sidebar holds the actions, and their screens open in
// the content area (plan/03). It starts on the login page; Esc from there goes Home.
func buildShell(app *tview.Application) *ui.Shell {
	sh := ui.NewShell(app)
	sh.Token = func() string { return appToken }

	sh.AddAction("Generate key", 'k', func() { keys.DoKeyGeneration(sh) }).
		AddAction("Register key", 'r', func() { keys.DoKeyRegistration(sh) }).
		AddAction("Key status", 'i', func() { keys.DoKeyStatus(sh) }).
		AddAction("Create wallet", 'w', func() { wallets.DoWalletGeneration(sh) }).
		AddAction("Generate address", 'a', func() { wallets.DoAddressGeneration(sh) }).
		AddAction("Set wallet context", 'c', func() { state.DoSetWalletAndAccountCtx(sh) }).
		AddAction("Get balance", 'b', func() { wallets.DoGetBalance(sh, state.WalletId, state.Address) }).
		AddAction("Send ETH", 's', func() { send.ShowSendForm(sh) }).
		AddAction("Help", '?', func() { sh.Show(ui.HelpPage, nil) }).
		AddAction("Log in again", 't', func() { sh.Show(ui.LoginPage, nil) }).
		AddAction("Quit", 'q', app.Stop)

	sh.SetHome(tview.NewTextView().SetText("\n  Choose an action from the menu on the left.\n\n  Press ? for help."))

	onLogin := func(email string) { keys.OnLogin(sh, email) }
	sh.Show(ui.LoginPage, token.GetTokenInputForm(sh, &appToken, onLogin))
	return sh
}

func main() {
	if err := buildShell(tview.NewApplication()).Run(); err != nil {
		panic(err)
	}
}
