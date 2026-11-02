package wallets

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func DoWalletGeneration(pages *tview.Pages, appToken string) {
	walletGenView := createWalletGenTextView(pages)
	walletGenView.Write([]byte("\nGenerating wallet...\n"))
	id, err := api.New(appToken).CreateWallet()
	if err != nil {
		walletGenView.Write([]byte("Error: " + ui.Describe(err, appToken) + "\n"))
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

// WalletAddressPair is one (wallet, address) entry from GET /api/v1/walletctx.
type WalletAddressPair = api.WalletAddressPair

// GetWalletsAndAddresses returns all the user's wallet/address pairs (empty if none).
func GetWalletsAndAddresses(appToken string) ([]WalletAddressPair, error) {
	return api.New(appToken).WalletContext()
}

func PairsToStrings(pairs []WalletAddressPair) []string {
	strings := make([]string, len(pairs))
	for i, v := range pairs {
		strings[i] = fmt.Sprintf("Wallet %6d | %s", v.WalletId, v.Address)
	}
	return strings
}
