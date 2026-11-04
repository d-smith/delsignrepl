package wallets

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/ui"
	"fmt"
	"math/big"
	"time"
)

// DoGetBalance fetches and shows the context address's balance in wei and ETH. The
// header shows it too.
func DoGetBalance(sh *ui.Shell) {
	if state.Address == "" {
		sh.Message("No address chosen yet. In Wallets (w), press Enter on the address to use.")
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

// RefreshBalance fetches the context address's balance into state. With no context it
// does nothing.
func RefreshBalance(token string) error {
	if state.Address == "" {
		return nil
	}
	b, err := getBalanceForAddress(state.Address, token)
	if err != nil {
		state.Balance, state.BalanceErr = nil, ui.Describe(err, token)
		return err
	}
	state.Balance, state.BalanceAt, state.BalanceErr = b, time.Now(), ""
	return nil
}
