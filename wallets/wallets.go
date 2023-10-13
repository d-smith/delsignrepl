package wallets

import (
	"delsignrepl/api"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func DoWalletGeneration(pages *tview.Pages, appToken string) {
	walletGenView := createWalletGenTextView(pages)
	walletGenView.Write([]byte("\nGenerating wallet...\n"))
	id, err := postWalletRequest(appToken)
	if err != nil {
		walletGenView.Write([]byte("Error: " + err.Error() + "\n"))
	} else {
		walletGenView.Write([]byte(fmt.Sprintf("Wallet ID: %d\n", id)))
	}

	walletGenView.Write([]byte("\nPress m to return to the main menu\n"))

	pages.AddPage("WalletGen", walletGenView, true, false)
	pages.SwitchToPage("WalletGen")
}

func createWalletGenTextView(pages *tview.Pages) *tview.TextView {
	textView := tview.NewTextView().SetText("Wallet generation")
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 109 {
			pages.SwitchToPage("Menu")
		}
		return event
	})

	return textView
}

func postWalletRequest(appToken string) (int, error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:3010/api/v1/wallets", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 201 {
		return 0, fmt.Errorf("error generating wallet")
	}

	var walletInfo api.WalletInfo
	err = json.NewDecoder(resp.Body).Decode(&walletInfo)
	if err != nil {
		return 0, err
	}

	return walletInfo.ID, nil
}

type WalletAddressPair struct {
	WalletId int    `json:"walletId"`
	Address  string `json:"address"`
}

func GetWalletsAndAddresses(appToken string) ([]WalletAddressPair, error) {
	var walletAddressPairs []WalletAddressPair
	req, err := http.NewRequest(http.MethodGet, "http://localhost:3010/api/v1/walletctx", nil)
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return walletAddressPairs, err
	}

	if resp.StatusCode != 200 {
		return walletAddressPairs, fmt.Errorf("error retrieving wallets and addresses for user")
	}

	err = json.NewDecoder(resp.Body).Decode(&walletAddressPairs)
	if err != nil {
		return walletAddressPairs, err
	}

	return walletAddressPairs, nil
}

func PairsToStrings(pairs []WalletAddressPair) []string {
	strings := make([]string, len(pairs))
	for i, v := range pairs {
		strings[i] = fmt.Sprintf("Wallet %6d | %s", v.WalletId, v.Address)
	}
	return strings
}
