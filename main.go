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

// Sidebar labels that buildShell highlights.
const (
	keysAction  = "Keys"
	loginAction = "Log in again"
)

// buildShell lays out the app: the sidebar holds the actions, their screens open in the
// content area, and the header and Home show the session (plan/03). It starts on the
// login page; Esc from there goes Home.
func buildShell(app *tview.Application) *ui.Shell {
	sh := ui.NewShell(app)
	sh.Token = func() string { return state.Token }
	sh.SetHeaderFunc(state.HeaderText)
	sh.SetHomeFunc(state.HomeText)

	sh.AddAction(keysAction, 'k', func() { keys.ShowKeys(sh) }).
		AddAction("Wallets", 'w', func() { wallets.ShowWallets(sh) }).
		AddAction("Get balance", 'b', func() { wallets.DoGetBalance(sh) }).
		AddAction("Send ETH", 's', func() { send.ShowSendForm(sh) }).
		AddAction("Activity", 'a', sh.ShowActivity).
		AddAction("Refresh", 'R', func() { refresh(sh) }).
		AddAction("Help", '?', func() { sh.Show(ui.HelpPage, nil) }).
		AddAction(loginAction, 'l', func() { sh.Show(ui.LoginPage, nil) }).
		AddAction("Quit", 'q', app.Stop)

	// The highlight shows where you are: on the login item while the login page is open,
	// then on Keys, the first step after logging in.
	onLogin := func(email string) {
		sh.HighlightAction(keysAction)
		keys.OnLogin(sh, email)
	}
	sh.Show(ui.LoginPage, token.GetTokenInputForm(sh, &state.Token, onLogin))
	sh.HighlightAction(loginAction)
	return sh
}

// refresh checks the key and fetches the balance again, for the header and Home.
func refresh(sh *ui.Shell) {
	if state.Email == "" {
		sh.Message("Log in first (l).")
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
