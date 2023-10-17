package main

import (
	"delsignrepl/keys"
	"delsignrepl/send"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/wallets"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var appToken string

func getMainList(pages *tview.Pages, app *tview.Application) *tview.List {
	menuList := tview.NewList().
		AddItem("Generate key", "Generate a key for signing API requests", 'k', nil).
		AddItem("Register key", "Register API signing key", 'r', nil).
		AddItem("Create wallet", "Create a wallet", 'w', nil).
		AddItem("Generate address", "Generate an address for a wallet", 'a', nil).
		AddItem("Set wallet/address context", "Set wallet/address context", 'c', nil).
		AddItem("Get balance", "Get balance for current wallet", 'b', nil).
		AddItem("Send ETH", "Send ETH from current wallet", 's', nil).
		AddItem("Quit", "Exit", 'q', nil)

	menuList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'k' {
			keys.DoKeyGeneration(pages)
		} else if event.Rune() == 'r' {
			keys.DoKeyRegistration(pages, appToken)
		} else if event.Rune() == 'q' {
			app.Stop()
		} else if event.Rune() == 'w' {
			wallets.DoWalletGeneration(pages, appToken)
		} else if event.Rune() == 'a' {
			wallets.DoAddressGeneration(pages, appToken)
		} else if event.Rune() == 'c' {
			state.DoSetWalletAndAccountCtx(pages, appToken)
		} else if event.Rune() == 'b' {
			wallets.DoGetBalance(pages, state.WalletId, state.Address, appToken)
		} else if event.Rune() == 's' {
			send.ShowSendForm(pages, appToken)
		}
		return event
	})

	return menuList
}

func createRegisterTextView(pages *tview.Pages) *tview.TextView {
	textView := tview.NewTextView().SetText("Key registration")
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 109 {
			pages.SwitchToPage("Menu")
		}
		return event
	})

	return textView
}

func main() {

	app := tview.NewApplication()

	var pages = tview.NewPages()

	list := getMainList(pages, app) //.SetBorder(true).SetTitle("Main list").SetTitleAlign(tview.AlignLeft)

	pages.AddPage("Menu", list, true, true)
	pages.AddPage("Add Token", token.GetTokenInputForm(pages, app, &appToken), true, true)
	//pages.AddPage("Keygen", createKeyGenTextView(pages), true, false)
	//pages.AddPage("Register", createRegisterTextView(pages), true, false)

	if err := app.SetRoot(pages, true).SetFocus(pages).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
