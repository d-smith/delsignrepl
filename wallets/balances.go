package wallets

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"fmt"
	"math/big"
)

// DoGetBalance fetches and shows the context address's balance in wei and ETH. The
// header shows it too.
func DoGetBalance(sh *ui.Shell) {
	if state.Address == "" {
		sh.Message("Wallet and address context not set. Choose one with Set wallet context (c).")
		return
	}
	if err := RefreshBalance(sh.Token()); err != nil {
		sh.Error("Error retrieving balance", err)
		return
	}
	sh.Message(fmt.Sprintf("Balance for wallet %d, address %s is %s wei (%s ETH)",
		state.WalletId, state.Address, state.Balance.String(), api.FormatEther(state.Balance)))
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
