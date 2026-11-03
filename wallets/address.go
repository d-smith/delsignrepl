package wallets

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"

	"github.com/rivo/tview"
)

// DoAddressGeneration lets the user pick a wallet and creates an address in it.
func DoAddressGeneration(sh *ui.Shell) {
	c := api.New(sh.Token())
	wallets, err := c.ListWallets()
	if err != nil {
		sh.Error("Error listing wallets", err)
		return
	}
	if len(wallets) == 0 {
		sh.Message("No wallets found. Create a wallet first (w).")
		return
	}

	selection := 0
	form := tview.NewForm().
		AddDropDown("Wallet ID: ", intsToStrings(wallets), 0, func(_ string, idx int) {
			selection = idx
		}).
		AddButton("Create address", func() {
			address, err := c.CreateAddress(wallets[selection])
			if err != nil {
				sh.Error("Error creating address", err)
				return
			}
			sh.Modal(fmt.Sprintf("New address for wallet %d is %s", wallets[selection], address),
				[]string{"OK"}, func(string) { sh.Home() })
		}).
		AddButton("Cancel", sh.Back)
	form.SetBorder(true).SetTitle(" New address ").SetTitleAlign(tview.AlignLeft)
	sh.Show("Address Gen", form)
}

func intsToStrings(ints []int) []string {
	strings := make([]string, len(ints))
	for i, v := range ints {
		strings[i] = fmt.Sprintf("%d", v)
	}
	return strings
}
