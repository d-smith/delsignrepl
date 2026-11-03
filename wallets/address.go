package wallets

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"

	"github.com/rivo/tview"
)

func DoAddressGeneration(pages *tview.Pages, appToken string) {
	wallets, err := api.New(appToken).ListWallets()
	if err != nil {
		ui.ShowError(pages, "Error listing wallets", err, appToken, "Menu")
		return
	}
	if len(wallets) == 0 {
		modal := tview.NewModal().
			SetText("No wallets found. Please create a wallet first").
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "OK" {
					pages.SwitchToPage("Menu")
				}
			})
		pages.AddPage("NoWallets", modal, true, false)
		pages.SwitchToPage("NoWallets")
		return
	}

	selection := 0
	form := tview.NewForm().
		AddDropDown("Select an option (hit Enter): ", intsToStrings(wallets), 0,
			func(val string, idx int) {
				selection = idx
			}).
		AddButton("Save", func() {
			address, err := api.New(appToken).CreateAddress(wallets[selection])
			if err != nil {
				ui.ShowError(pages, "Error creating address for wallet", err, appToken, "Menu")
				return
			}
			modal := tview.NewModal().
				SetText(fmt.Sprintf("New address for wallet %d is %s",
					wallets[selection], address)).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "OK" {
						pages.SwitchToPage("Menu")
					}
				})

			pages.AddPage("NewAddress", modal, true, false)
			pages.SwitchToPage("NewAddress")
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("Menu")
		})
	form.SetBorder(true).SetTitle("Select wallet for address gen").SetTitleAlign(tview.AlignLeft)
	pages.AddPage("Address Gen", form, true, false)
	pages.SwitchToPage("Address Gen")
}

func intsToStrings(ints []int) []string {
	strings := make([]string, len(ints))
	for i, v := range ints {
		strings[i] = fmt.Sprintf("%d", v)
	}
	return strings
}
