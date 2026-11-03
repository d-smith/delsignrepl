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
			ui.ShowError(pages, "Error retrieving balance", err, appToken, "Menu")
			return
		} else {
			msg = fmt.Sprintf("Balance for wallet %d, address %s is %s wei (%s ETH)",
				walletId, address, balance.String(), api.FormatEther(balance))
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
	amount, err := api.ParseWei(b.Amount)
	if err != nil {
		return nil, fmt.Errorf("server sent an invalid amount: %w", err)
	}
	return amount, nil
}
