package wallets

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rivo/tview"
)

func DoAddressGeneration(pages *tview.Pages, appToken string) {
	wallets, _ := getWallets(appToken)
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
			var modal *tview.Modal
			address, err := createWalletAddress(wallets[selection], appToken)
			if err != nil {
				modal = tview.NewModal().
					SetText(fmt.Sprintf("Error creating address for wallet: %s",
						err.Error())).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonLabel == "OK" {
							pages.SwitchToPage("Menu")
						}
					})
			} else {

				modal = tview.NewModal().
					SetText(fmt.Sprintf("New address for wallet %d is %s",
						wallets[selection], address)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonLabel == "OK" {
							pages.SwitchToPage("Menu")
						}
					})
			}

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

func getWallets(appToken string) ([]int, error) {
	var wallets []int
	req, err := http.NewRequest(http.MethodGet, "http://localhost:3010/api/v1/wallets", nil)
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return wallets, err
	}

	if resp.StatusCode != 201 {
		return wallets, fmt.Errorf("error generating wallet")
	}

	err = json.NewDecoder(resp.Body).Decode(&wallets)
	if err != nil {
		return wallets, err
	}

	return wallets, nil
}

func createWalletAddress(walletId int, appToken string) (string, error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:3010/api/v1/wallets/"+fmt.Sprintf("%d", walletId)+"/addresses", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("error generating wallet address")
	}

	var eoa struct {
		Address string `json:"eoa"`
	}

	err = json.NewDecoder(resp.Body).Decode(&eoa)
	if err != nil {
		return "", err
	}

	return eoa.Address, nil
}

func intsToStrings(ints []int) []string {
	strings := make([]string, len(ints))
	for i, v := range ints {
		strings[i] = fmt.Sprintf("%d", v)
	}
	return strings
}
