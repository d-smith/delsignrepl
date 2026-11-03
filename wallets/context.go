package wallets

import (
	"delsignrepl/state"
	"delsignrepl/ui"
	"time"

	"github.com/rivo/tview"
)

// DoSetContext lets the user pick the wallet and address that balance and send use, then
// fetches its balance for the header.
func DoSetContext(sh *ui.Shell) {
	pairs, err := GetWalletsAndAddresses(sh.Token())
	if err != nil {
		sh.Error("Error retrieving wallet and address context", err)
		return
	}
	if len(pairs) == 0 {
		sh.Message("No wallets with addresses found. Create a wallet (w) and an address (a) first.")
		return
	}

	selection := 0
	form := tview.NewForm().
		AddDropDown("Wallet | address: ", PairsToStrings(pairs), 0, func(_ string, idx int) {
			selection = idx
		}).
		AddButton("Use this", func() {
			state.SetContext(pairs[selection].WalletId, pairs[selection].Address)
			sh.Home()
			if err := RefreshBalance(sh.Token()); err != nil {
				sh.Error("Context set, but fetching the balance failed", err)
			}
		}).
		AddButton("Cancel", sh.Back)
	form.SetBorder(true).SetTitle(" Wallet and address for balance and send ").SetTitleAlign(tview.AlignLeft)
	sh.Show("SetCtx", form)
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
