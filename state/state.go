package state

import (
	"crypto/rsa"
	"delsignrepl/wallets"
	"fmt"

	"github.com/rivo/tview"
)

var PrivateKey *rsa.PrivateKey
var PublicKeyDER string
var WalletId int
var Address string

func DoSetWalletAndAccountCtx(pages *tview.Pages, appToken string) {
	selection := 0
	walletAddressPairs, err := wallets.GetWalletsAndAddresses(appToken)

	if err != nil {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Error retrieving wallet and address context: %s", err.Error())).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(_ int, _ string) {
				pages.SwitchToPage("Menu")
			})

		pages.AddPage("CtxErr", modal, true, false)
		pages.SwitchToPage("CtxErr")
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
