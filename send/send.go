package send

import (
	"bytes"
	"delsignrepl/keys"
	"delsignrepl/state"
	"delsignrepl/wallets"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/rivo/tview"
)

func handleContextNotSet(pages *tview.Pages) {
	modal := tview.NewModal().
		SetText("Wallet and address context not set").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.SwitchToPage("Menu")
		})

	pages.AddPage("CtxErr", modal, true, false)
	pages.SwitchToPage("CtxErr")
}

func handleNoWallets(pages *tview.Pages) {
	modal := tview.NewModal().
		SetText("No eligible addresses associated with this wallet to use as destination").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(_ int, _ string) {
			pages.SwitchToPage("SendEth")
		})

	pages.AddPage("NoAddresses", modal, true, false)
	pages.SwitchToPage("NoAddresses")
}

func filterWalletAddressPairs(pairs []wallets.WalletAddressPair, address string) []wallets.WalletAddressPair {
	var filtered []wallets.WalletAddressPair
	for _, pair := range pairs {
		if pair.Address != address {
			filtered = append(filtered, pair)
		}
	}
	return filtered
}

func ShowSendForm(pages *tview.Pages, appToken string) {
	if state.Address == "" {
		handleContextNotSet(pages)
		return
	}

	destination := tview.NewInputField().
		SetLabel("Destination address: ").
		SetFieldWidth(60).
		SetAcceptanceFunc(tview.InputFieldMaxLength(60))

	var address string
	var selection int

	walletAddress := tview.NewCheckbox().SetLabel("Use Wallet Address").SetChecked(false)
	walletAddress.SetChangedFunc(func(checked bool) {
		if checked {
			walletAddressPairs, err := wallets.GetWalletsAndAddresses(appToken)

			if err != nil {
				modal := tview.NewModal().
					SetText(fmt.Sprintf("Error retrieving wallet and address context: %s", err.Error())).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(_ int, _ string) {
						pages.SwitchToPage("SendEth")
					})

				pages.AddPage("CtxErr", modal, true, false)
				pages.SwitchToPage("CtxErr")
				return
			}

			walletAddressPairs = filterWalletAddressPairs(walletAddressPairs, state.Address)
			if len(walletAddressPairs) == 0 {
				handleNoWallets(pages)
				return
			}

			form := tview.NewForm().
				AddButton("Done", func() {
					address = walletAddressPairs[selection].Address
					destination.SetText(address)
					pages.SwitchToPage("SendEth")
				}).
				AddButton("Cancel", func() {
					pages.SwitchToPage("SendEth")
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

			pages.AddPage("WalletAddress", form, true, false)
			pages.SwitchToPage("WalletAddress")

		} else {
			destination.SetText("")
		}
	})

	amount := tview.NewInputField().
		SetLabel("Amount (in Wei): ").
		SetFieldWidth(60).
		SetAcceptanceFunc(tview.InputFieldInteger)

	source := tview.NewInputField().
		SetLabel("Source address: ").
		SetFieldWidth(60).
		SetText(state.Address).
		SetAcceptanceFunc(func(s string, lc rune) bool {
			return false
		})

	form := tview.NewForm().
		//AddTextView("Source Address", state.Address, 0, 0, false, false).
		AddFormItem(source).
		AddFormItem(destination).
		AddFormItem(walletAddress).
		AddFormItem(amount).
		AddButton("Send", func() {
			processSendForm(pages, destination.GetText(), amount.GetText(), appToken)
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("Menu")
		})
	form.SetFocus(1)
	form.SetBorder(true).SetTitle("Send ETH").SetTitleAlign(tview.AlignLeft)

	pages.AddPage("SendEth", form, true, false)
	pages.SwitchToPage("SendEth")
}

func processSendForm(pages *tview.Pages, destination string, amount string, appToken string) {
	var modal *tview.Modal

	txnid, err := sendEth(destination, amount, appToken) // Source is from wallet context
	if err != nil {
		modal = tview.NewModal().
			SetText(fmt.Sprintf("Error sending ETH: %s",
				err.Error())).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "OK" {
					pages.SwitchToPage("Menu")
				}
			})
	} else {

		modal = tview.NewModal().
			SetText(fmt.Sprintf("Eth send - transaction id: %s", txnid)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "OK" {
					pages.SwitchToPage("Menu")
				}
			})
	}

	pages.AddPage("NewAddress", modal, true, false)
	pages.SwitchToPage("NewAddress")
}

type SendPayload struct {
	SourceAddress      string   `json:"source"`
	DestinationAddress string   `json:"dest"`
	Amount             *big.Int `json:"amount"`
	Signature          string   `json:"sig"`
}

func formSendEthPayload(destination string, amount string) (*SendPayload, error) {
	amountInt := new(big.Int)
	amountInt.SetString(amount, 10)

	msg := fmt.Sprintf("%s%s%d", state.Address, destination, amountInt)

	if state.PrivateKey == nil {
		return nil, fmt.Errorf("private key not set")
	}
	sig := keys.Sign(msg, state.PrivateKey)

	return &SendPayload{
		SourceAddress:      state.Address,
		DestinationAddress: destination,
		Amount:             amountInt,
		Signature:          sig,
	}, nil
}

func sendEth(destination string, amount string, appToken string) (string, error) {
	payload, err := formSendEthPayload(destination, amount)
	if err != nil {
		return "", err
	}

	payloadJson, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	bodyReader := bytes.NewReader(payloadJson)
	req, err := http.NewRequest(http.MethodPost, "http://localhost:3010/api/v1/wallets/send", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+appToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("error sending ETH: %s", resp.Status)
	}

	var transactionContext struct {
		Txnid string `json:"txnid"`
	}

	err = json.NewDecoder(resp.Body).Decode(&transactionContext)
	if err != nil {
		return "", err
	}

	return transactionContext.Txnid, nil

}
