package wallets

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"
	"math/big"
)

// DoGetBalance shows the balance of the context address in wei and ETH.
func DoGetBalance(sh *ui.Shell, walletId int, address string) {
	if walletId == 0 || address == "" {
		sh.Message("Wallet and address context not set. Choose one with Set wallet context (c).")
		return
	}
	balance, err := getBalanceForAddress(address, sh.Token())
	if err != nil {
		sh.Error("Error retrieving balance", err)
		return
	}
	sh.Message(fmt.Sprintf("Balance for wallet %d, address %s is %s wei (%s ETH)",
		walletId, address, balance.String(), api.FormatEther(balance)))
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
