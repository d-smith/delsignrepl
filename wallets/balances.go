package wallets

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/rivo/tview"
)

func DoGetBalance(pages *tview.Pages, walletId int, address string, appToken string) {
	var msg string
	if walletId == 0 || address == "" {
		msg = "Wallet and address context not set"
	} else {

		balance, err := getBalanceForAddress(address, appToken)
		if err != nil {
			msg = "Error retrieving balance: " + err.Error()
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
