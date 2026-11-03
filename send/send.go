package send

import (
	"delsignrepl/api"
	"delsignrepl/keys"
	"delsignrepl/state"
	"delsignrepl/ui"
	"delsignrepl/wallets"
	"errors"
	"fmt"
	"math/big"
	"strings"

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

// ShowSendForm shows an empty send form for the current wallet context.
func ShowSendForm(pages *tview.Pages, appToken string) {
	showSendForm(pages, appToken, "", "")
}

// showSendForm shows the send form with dest and amount filled in ("Edit and send again").
func showSendForm(pages *tview.Pages, appToken, destText, amountText string) {
	if state.Address == "" {
		handleContextNotSet(pages)
		return
	}

	destination := tview.NewInputField().
		SetLabel("Destination address: ").
		SetFieldWidth(60).
		SetText(destText).
		SetAcceptanceFunc(tview.InputFieldMaxLength(60))

	var address string
	var selection int

	walletAddress := tview.NewCheckbox().SetLabel("Use Wallet Address").SetChecked(false)
	walletAddress.SetChangedFunc(func(checked bool) {
		if checked {
			walletAddressPairs, err := wallets.GetWalletsAndAddresses(appToken)

			if err != nil {
				ui.ShowError(pages, "Error retrieving wallet and address context", err, appToken, "SendEth")
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
		SetLabel("Amount (in wei): ").
		SetFieldWidth(80).
		SetText(amountText).
		SetAcceptanceFunc(weiDigitsOnly)

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
			processSendForm(pages, appToken, destination.GetText(), amount.GetText())
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("Menu")
		})
	form.SetFocus(1)
	form.SetBorder(true).SetTitle("Send ETH").SetTitleAlign(tview.AlignLeft)

	pages.AddPage("SendEth", form, true, false)
	pages.SwitchToPage("SendEth")
}

// weiDigitsOnly lets the amount field hold decimal digits only, up to the wire format's
// 78 characters. Leading zeros are caught by api.ParseWei on Send.
func weiDigitsOnly(text string, _ rune) bool {
	if len(text) > 78 {
		return false
	}
	for _, c := range text {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// validateSend checks the form's values before anything is signed.
func validateSend(source, destText, amountText string) (dest string, wei *big.Int, err error) {
	dest = strings.TrimSpace(destText)
	if !api.ValidAddress(dest) {
		return "", nil, fmt.Errorf("destination %q isn't an address: use 0x followed by 40 hex digits", dest)
	}
	if !api.ValidAddress(source) {
		return "", nil, fmt.Errorf("source %q isn't an address; set the wallet context again", source)
	}
	if amountText == "" {
		return "", nil, errors.New("enter an amount in wei")
	}
	wei, err = api.ParseWei(amountText)
	if err != nil {
		return "", nil, err
	}
	if wei.Sign() == 0 {
		return "", nil, errors.New("the amount must be more than 0 wei")
	}
	return dest, wei, nil
}

func showModal(pages *tview.Pages, name, text string, buttons []string, done func(label string)) {
	modal := tview.NewModal().
		SetText(text).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, label string) { done(label) })
	pages.AddPage(name, modal, true, false)
	pages.SwitchToPage(name)
}

const (
	btnConfirm = "Confirm"
	btnCancel  = "Cancel"
	btnMenu    = "Menu"
	btnRetry   = "Retry same request"
	btnBalance = "Check balance"
	btnEdit    = "Edit and send again"
)

func processSendForm(pages *tview.Pages, appToken, destText, amountText string) {
	source := state.Address
	dest, wei, err := validateSend(source, destText, amountText)
	if err != nil {
		showModal(pages, "SendInvalid", err.Error(), []string{"OK"}, func(string) {
			pages.SwitchToPage("SendEth")
		})
		return
	}
	if state.PrivateKey == nil {
		showModal(pages, "SendInvalid", "No signing key loaded. Generate and register a key first.",
			[]string{"OK"}, func(string) { pages.SwitchToPage("Menu") })
		return
	}

	// Plan D8: confirm first; the request is built and signed only on Confirm.
	text := fmt.Sprintf("Send %s wei (%s ETH)\n\nfrom %s\nto   %s?",
		wei.String(), api.FormatEther(wei), source, dest)
	showModal(pages, "SendConfirm", text, []string{btnConfirm, btnCancel}, func(label string) {
		if label != btnConfirm {
			pages.SwitchToPage("SendEth")
			return
		}
		flow := NewFlow(api.New(appToken), state.PrivateKey)
		showResult(pages, appToken, flow, flow.Confirm(source, dest, wei), dest, amountText)
	})
}

// showResult shows one attempt's outcome with the buttons plan §3 lists for it.
func showResult(pages *tview.Pages, appToken string, flow *Flow, r Result, dest, amountText string) {
	toMenu := func() { pages.SwitchToPage("Menu") }
	checkBalance := func() { wallets.DoGetBalance(pages, state.WalletId, state.Address, appToken) }
	edit := func() { showSendForm(pages, appToken, dest, amountText) }

	switch r.Outcome {
	case Sent:
		showModal(pages, "SendResult",
			fmt.Sprintf("Broadcast. Transaction %s.\n\nThe balance changes once it's mined.", r.Txnid),
			[]string{"OK"}, func(string) { toMenu() })

	case NoResponse:
		text := fmt.Sprintf("No response: %v\n\nThe transfer may or may not have happened.", r.Err)
		buttons := []string{btnBalance, btnMenu}
		if flow.CanRetry() {
			buttons = append([]string{btnRetry}, buttons...)
		} else {
			text += "\n\nThis request has expired; check the balance before sending again."
		}
		showModal(pages, "SendResult", text, buttons, func(label string) {
			switch label {
			case btnRetry:
				next, err := flow.Retry()
				if err != nil {
					// Expired while the modal was open: say so, never resend.
					showResult(pages, appToken, flow, Result{Outcome: NoResponse, Err: r.Err}, dest, amountText)
					return
				}
				showResult(pages, appToken, flow, next, dest, amountText)
			case btnBalance:
				checkBalance()
			default:
				toMenu()
			}
		})

	case AlreadyProcessed:
		showModal(pages, "SendResult",
			"The server already processed this request. The transfer may have happened.\n\n"+
				"Check the balance before sending again.",
			[]string{btnBalance, btnMenu}, func(label string) {
				if label == btnBalance {
					checkBalance()
					return
				}
				toMenu()
			})

	case NodeError:
		showModal(pages, "SendResult",
			fmt.Sprintf("Ethereum node error: %s\n\nThe transfer was most likely not sent. "+
				"That request can't be reused.", errText(r.Err)),
			[]string{btnEdit, btnMenu}, func(label string) {
				if label == btnEdit {
					edit()
					return
				}
				toMenu()
			})

	case SignatureRejected:
		var status string
		if st, err := keys.CheckKey(api.New(appToken), keys.StoreFromEnv(), state.Email); err == nil {
			status = "Key status: " + st.String()
		} else {
			status = "couldn't check the key: " + ui.Describe(err, appToken)
		}
		showModal(pages, "SendResult",
			"Signature rejected. Your key may not be the registered one.\n\n"+status,
			[]string{"OK"}, func(string) { toMenu() })

	case Rejected:
		showModal(pages, "SendResult",
			fmt.Sprintf("Request rejected: %s", errText(r.Err)),
			[]string{btnEdit, btnMenu}, func(label string) {
				if label == btnEdit {
					edit()
					return
				}
				toMenu()
			})

	default:
		ui.ShowError(pages, "Error sending ETH", r.Err, appToken, "Menu")
	}
}

// errText is the server's text for an api.Error, or the error itself.
func errText(err error) string {
	var e *api.Error
	if errors.As(err, &e) && e.Text != "" {
		return e.Text
	}
	return err.Error()
}
