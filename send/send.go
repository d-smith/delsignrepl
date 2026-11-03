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
func ShowSendForm(sh *ui.Shell) {
	showSendForm(sh, "", "")
}

// showSendForm shows the send form with dest and amount filled in ("Edit and send again").
func showSendForm(sh *ui.Shell, destText, amountText string) {
	if state.Address == "" {
		sh.Message("Wallet and address context not set. Choose one with Set wallet context (c).")
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
			walletAddressPairs, err := wallets.GetWalletsAndAddresses(sh.Token())
			if err != nil {
				sh.Error("Error retrieving wallet and address context", err)
				return
			}

			walletAddressPairs = filterWalletAddressPairs(walletAddressPairs, state.Address)
			if len(walletAddressPairs) == 0 {
				sh.Message("You have no other addresses to send to.")
				return
			}

			form := tview.NewForm().
				AddDropDown("Wallet | address: ", wallets.PairsToStrings(walletAddressPairs), 0,
					func(_ string, idx int) {
						selection = idx
					}).
				AddButton("Use this", func() {
					address = walletAddressPairs[selection].Address
					destination.SetText(address)
					sh.Back()
				}).
				AddButton("Cancel", sh.Back)
			form.SetBorder(true).SetTitle(" Send to one of my addresses ").SetTitleAlign(tview.AlignLeft)
			sh.Show("WalletAddress", form)

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
			processSendForm(sh, destination.GetText(), amount.GetText())
		}).
		AddButton("Cancel", sh.Back)
	form.SetFocus(1)
	form.SetBorder(true).SetTitle(" Send ETH ").SetTitleAlign(tview.AlignLeft)
	sh.Show("SendEth", form)
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

const (
	btnConfirm = "Confirm"
	btnCancel  = "Cancel"
	btnHome    = "Home"
	btnRetry   = "Retry same request"
	btnBalance = "Check balance"
	btnEdit    = "Edit and send again"
)

func processSendForm(sh *ui.Shell, destText, amountText string) {
	source := state.Address
	dest, wei, err := validateSend(source, destText, amountText)
	if err != nil {
		sh.Message(err.Error()) // OK returns to the form
		return
	}
	if state.PrivateKey == nil {
		sh.Modal("No signing key loaded. Generate (k) and register (r) a key first.",
			[]string{"OK"}, func(string) { sh.Home() })
		return
	}

	// Plan D8: confirm first; the request is built and signed only on Confirm.
	text := fmt.Sprintf("Send %s wei (%s ETH)\n\nfrom %s\nto   %s?",
		wei.String(), api.FormatEther(wei), source, dest)
	sh.Modal(text, []string{btnConfirm, btnCancel}, func(label string) {
		if label != btnConfirm {
			return // back to the form
		}
		flow := NewFlow(api.New(sh.Token()), state.PrivateKey)
		showResult(sh, flow, flow.Confirm(source, dest, wei), dest, amountText)
	})
}

// showResult shows one attempt's outcome with the buttons plan §3 lists for it.
func showResult(sh *ui.Shell, flow *Flow, r Result, dest, amountText string) {
	toMenu := sh.Home
	checkBalance := func() {
		sh.Home()
		wallets.DoGetBalance(sh, state.WalletId, state.Address)
	}
	edit := func() { showSendForm(sh, dest, amountText) }

	switch r.Outcome {
	case Sent:
		sh.Modal(
			fmt.Sprintf("Broadcast. Transaction %s.\n\nThe balance changes once it's mined.", r.Txnid),
			[]string{"OK"}, func(string) { toMenu() })

	case NoResponse:
		text := fmt.Sprintf("No response: %v\n\nThe transfer may or may not have happened.", r.Err)
		buttons := []string{btnBalance, btnHome}
		if flow.CanRetry() {
			buttons = append([]string{btnRetry}, buttons...)
		} else {
			text += "\n\nThis request has expired; check the balance before sending again."
		}
		sh.Modal(text, buttons, func(label string) {
			switch label {
			case btnRetry:
				next, err := flow.Retry()
				if err != nil {
					// Expired while the modal was open: say so, never resend.
					showResult(sh, flow, Result{Outcome: NoResponse, Err: r.Err}, dest, amountText)
					return
				}
				showResult(sh, flow, next, dest, amountText)
			case btnBalance:
				checkBalance()
			default:
				toMenu()
			}
		})

	case AlreadyProcessed:
		sh.Modal(
			"The server already processed this request. The transfer may have happened.\n\n"+
				"Check the balance before sending again.",
			[]string{btnBalance, btnHome}, func(label string) {
				if label == btnBalance {
					checkBalance()
					return
				}
				toMenu()
			})

	case NodeError:
		sh.Modal(
			fmt.Sprintf("Ethereum node error: %s\n\nThe transfer was most likely not sent. "+
				"That request can't be reused.", errText(r.Err)),
			[]string{btnEdit, btnHome}, func(label string) {
				if label == btnEdit {
					edit()
					return
				}
				toMenu()
			})

	case SignatureRejected:
		var status string
		if st, err := keys.CheckKey(api.New(sh.Token()), keys.StoreFromEnv(), state.Email); err == nil {
			status = "Key status: " + st.String()
		} else {
			status = "couldn't check the key: " + ui.Describe(err, sh.Token())
		}
		sh.Modal(
			"Signature rejected. Your key may not be the registered one.\n\n"+status,
			[]string{"OK"}, func(string) { toMenu() })

	case Rejected:
		sh.Modal(
			fmt.Sprintf("Request rejected: %s", errText(r.Err)),
			[]string{btnEdit, btnHome}, func(label string) {
				if label == btnEdit {
					edit()
					return
				}
				toMenu()
			})

	default:
		sh.Error("Error sending ETH", r.Err) // OK returns to the form
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
