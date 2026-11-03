package state

import (
	"crypto/rsa"
	"delsignrepl/ui"
	"delsignrepl/wallets"

	"github.com/rivo/tview"
)

// Email is the logged-in user's email (from the token). PrivateKey is their current
// signing key, loaded from the key store, or nil if there isn't one yet.
var Email string
var PrivateKey *rsa.PrivateKey
var WalletId int
var Address string

// DoSetWalletAndAccountCtx lets the user pick the wallet and address that balance and
// send use.
func DoSetWalletAndAccountCtx(sh *ui.Shell) {
	pairs, err := wallets.GetWalletsAndAddresses(sh.Token())
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
		AddDropDown("Wallet | address: ", wallets.PairsToStrings(pairs), 0, func(_ string, idx int) {
			selection = idx
		}).
		AddButton("Use this", func() {
			WalletId = pairs[selection].WalletId
			Address = pairs[selection].Address
			sh.Home()
		}).
		AddButton("Cancel", sh.Back)
	form.SetBorder(true).SetTitle(" Wallet and address for balance and send ").SetTitleAlign(tview.AlignLeft)
	sh.Show("SetCtx", form)
}
