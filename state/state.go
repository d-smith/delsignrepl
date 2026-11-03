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

func DoSetWalletAndAccountCtx(pages *tview.Pages, appToken string) {
	selection := 0
	walletAddressPairs, err := wallets.GetWalletsAndAddresses(appToken)

	if err != nil {
		ui.ShowError(pages, "Error retrieving wallet and address context", err, appToken, "Menu")
		return
	}

	if len(walletAddressPairs) == 0 {
		modal := tview.NewModal().
			SetText("No wallets and addresses found. Please create a wallet and address first").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				pages.SwitchToPage("Menu")
			})

		pages.AddPage("NoWallets", modal, true, false)
		pages.SwitchToPage("NoWallets")
		return
	}

	form := tview.NewForm().
		AddButton("Done", func() {
			WalletId = walletAddressPairs[selection].WalletId
			Address = walletAddressPairs[selection].Address
			pages.SwitchToPage("Menu")
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("Menu")
		})

	if err == nil {

		form.
			AddDropDown("Select an option (hit Enter): ", wallets.PairsToStrings(walletAddressPairs), 0,
				func(val string, idx int) {
					selection = idx
				})
	} else {
		form.AddTextView("Status", err.Error(), 50, 1, true, true)

	}

	form.SetBorder(true).SetTitle("Select wallet and address for txn context").SetTitleAlign(tview.AlignLeft)
	pages.AddPage("SetCtx", form, true, false)
	pages.SwitchToPage("SetCtx")
}
