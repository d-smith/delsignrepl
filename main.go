package main

import (
	"delsignrepl/keys"
	"delsignrepl/send"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/wallets"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var appToken string

func getBalanceForAddress(address string) (*big.Int, error) {

	req, err := http.NewRequest(http.MethodGet,
		"http://localhost:3010/api/v1/wallets/balance/"+address, nil)
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error retrieving wallets and addresses for user")
	}

	var accountBalance struct {
		Address string   `json:"address"`
		Amount  *big.Int `json:"amount"`
	}

	err = json.NewDecoder(resp.Body).Decode(&accountBalance)
	if err != nil {
		return nil, err
	}

	return accountBalance.Amount, nil
}

func weiToEther(val *big.Int) *big.Int {
	return new(big.Int).Div(val, big.NewInt(1e18))
}

func doGetBalance(pages *tview.Pages) {
	var msg string
	if state.WalletId == 0 || state.Address == "" {
		msg = "Wallet and address context not set"
	} else {

		balance, err := getBalanceForAddress(state.Address)
		if err != nil {
			msg = "Error retrieving balance: " + err.Error()
		} else {
			msg = fmt.Sprintf("Balance for wallet %d, address %s is %d wei (%d ether)",
				state.WalletId, state.Address, balance, weiToEther(balance))
		}

	}

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				pages.SwitchToPage("Menu")
			}

		})

	pages.AddPage("Balance", modal, true, false)
	pages.SwitchToPage("Balance")

}

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
			doGetBalance(pages)
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
