package main

import (
	"delsignrepl/keys"
	"delsignrepl/send"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/ui"
	"delsignrepl/wallets"
	"time"

	"github.com/rivo/tview"
)

// buildShell lays out the app: the sidebar holds the actions, their screens open in the
// content area, and the header and Home show the session (plan/03). It starts on the
// login page; Esc from there goes Home.
func buildShell(app *tview.Application) *ui.Shell {
	sh := ui.NewShell(app)
	sh.Token = func() string { return state.Token }
	sh.SetHeaderFunc(state.HeaderText)
	sh.SetHomeFunc(state.HomeText)

	sh.AddAction("Generate key", 'k', func() { keys.DoKeyGeneration(sh) }).
		AddAction("Register key", 'r', func() { keys.DoKeyRegistration(sh) }).
		AddAction("Key status", 'i', func() { keys.DoKeyStatus(sh) }).
		AddAction("Create wallet", 'w', func() { wallets.DoWalletGeneration(sh) }).
		AddAction("Generate address", 'a', func() { wallets.DoAddressGeneration(sh) }).
		AddAction("Set wallet context", 'c', func() { wallets.DoSetContext(sh) }).
		AddAction("Get balance", 'b', func() { wallets.DoGetBalance(sh) }).
		AddAction("Send ETH", 's', func() { send.ShowSendForm(sh) }).
		AddAction("Refresh", 'R', func() { refresh(sh) }).
		AddAction("Help", '?', func() { sh.Show(ui.HelpPage, nil) }).
		AddAction("Log in again", 't', func() { sh.Show(ui.LoginPage, nil) }).
		AddAction("Quit", 'q', app.Stop)

	onLogin := func(email string) { keys.OnLogin(sh, email) }
	sh.Show(ui.LoginPage, token.GetTokenInputForm(sh, &state.Token, onLogin))
	return sh
}

// refresh checks the key and fetches the balance again, for the header and Home.
func refresh(sh *ui.Shell) {
	if state.Email == "" {
		sh.Message("Log in first (t).")
		return
	}
	if _, err := keys.RefreshKeyStatus(state.Token); err != nil {
		sh.Error("Error checking your key", err)
		return
	}
	if err := wallets.RefreshBalance(state.Token); err != nil {
		sh.Error("Error retrieving balance", err)
		return
	}
	sh.Status("Refreshed at " + time.Now().Format("15:04:05"))
}

func main() {
	if err := buildShell(tview.NewApplication()).Run(); err != nil {
		panic(err)
	}
}
