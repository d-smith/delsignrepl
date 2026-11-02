package wallets

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"
	"math/big"

	"github.com/rivo/tview"
)

func DoGetBalance(pages *tview.Pages, walletId int, address string, appToken string) {
	var msg string
	if walletId == 0 || address == "" {
		msg = "Wallet and address context not set"
	} else {

		balance, err := getBalanceForAddress(address, appToken)
		if err != nil {
			msg = "Error retrieving balance: " + ui.Describe(err, appToken)
		} else {
			msg = fmt.Sprintf("Balance for wallet %d, address %s is %d wei (%d ether)",
				walletId, address, balance, weiToEther(balance))
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

func getBalanceForAddress(address string, appToken string) (*big.Int, error) {
	b, err := api.New(appToken).Balance(address)
	if err != nil {
		return nil, err
	}
	// Amounts are decimal strings on the wire. Phase 4 replaces this with api.ParseWei.
	amount, ok := new(big.Int).SetString(b.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("server sent an invalid amount %q", b.Amount)
	}
	return amount, nil
}

func weiToEther(val *big.Int) *big.Int {
	return new(big.Int).Div(val, big.NewInt(1e18))
}
