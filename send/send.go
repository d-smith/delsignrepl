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

// SendPage is the send form's page name.
const SendPage = "SendEth"

// ShowSendForm shows an empty send form for the current wallet context.
func ShowSendForm(sh *ui.Shell) {
	showSendForm(sh, "", "")
}

// showSendForm shows the send form with dest and amount filled in ("Edit and send again").
func showSendForm(sh *ui.Shell, destText, amountText string) {
	if state.Address == "" {
		sh.Message("No address chosen yet. In Wallets (w), press Enter on the address to send from.")
		return
	}

	// Text views in a form are skipped by Tab only if they aren't scrollable.
	source := tview.NewTextView().SetLabel("From: ").SetText(state.Address).SetSize(1, 0).SetScrollable(false)
	destination := tview.NewInputField().
		SetLabel("To: ").
		SetFieldWidth(44).
		SetPlaceholder("0x followed by 40 hex digits").
		SetText(destText).
		SetAcceptanceFunc(tview.InputFieldMaxLength(60))
	amount := tview.NewInputField().
		SetLabel("Amount (wei): ").
		SetFieldWidth(80).
		SetPlaceholder("1000000000000000 is 0.001 ETH").
		SetText(amountText).
		SetAcceptanceFunc(weiDigitsOnly)
	preview := tview.NewTextView().SetLabel(" ").SetSize(1, 0).SetScrollable(false).
		SetText(amountPreview(amountText))
	amount.SetChangedFunc(func(text string) { preview.SetText(amountPreview(text)) })

	form := tview.NewForm().
		AddFormItem(source).
		AddFormItem(ui.TrimPaste(destination)).
		AddFormItem(ui.TrimPaste(amount)).
		AddFormItem(preview).
		AddButton("Send", func() {
			processSendForm(sh, destination.GetText(), amount.GetText())
		}).
		AddButton("Pick from my addresses", func() {
			wallets.PickAddress(sh, state.Address, func(a string) { destination.SetText(a) })
		}).
		AddButton("Cancel", sh.Back)
	form.SetFocus(1)
	form.SetBorder(true).SetTitle(" Send ETH ").SetTitleAlign(tview.AlignLeft)
	sh.Show(SendPage, form)
}

// amountPreview is the line under the amount: its value in ETH as it's typed, or what's
// wrong with it.
func amountPreview(text string) string {
	if text == "" {
		return ""
	}
	wei, err := api.ParseWei(text)
	switch {
	case err != nil && len(text) > 1 && text[0] == '0':
		return "no leading zeros, please"
	case err != nil:
		return "more than the largest amount (2^256-1 wei)"
	case wei.Sign() == 0:
		return "= 0 ETH (the amount must be more than 0)"
	}
	return "= " + api.FormatEther(wei) + " ETH"
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
	btnCopyTx  = "Copy txnid"
)

func processSendForm(sh *ui.Shell, destText, amountText string) {
	source := state.Address
	dest, wei, err := validateSend(source, destText, amountText)
	if err != nil {
		sh.Message(err.Error()) // OK returns to the form
		return
	}
	if state.PrivateKey == nil {
		sh.Modal("No signing key loaded. Open Keys (k) to generate and register one first.",
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
		r := flow.Confirm(source, dest, wei)
		logAttempt(sh, flow, r, source, dest, wei, false)
		showResult(sh, flow, r, dest, amountText)
	})
}

// showResult shows one attempt's outcome with the buttons plan §3 lists for it.
func showResult(sh *ui.Shell, flow *Flow, r Result, dest, amountText string) {
	toMenu := sh.Home
	checkBalance := func() {
		sh.Home()
		wallets.DoGetBalance(sh)
	}
	edit := func() { showSendForm(sh, dest, amountText) }

	switch r.Outcome {
	case Sent:
		// Anvil mines at once, so the header's balance can catch up now. A failure only
		// leaves it saying "?".
		_ = wallets.RefreshBalance(sh.Token())
		sh.Modal(
			fmt.Sprintf("Broadcast. Transaction %s.\n\nThe balance changes once it's mined. "+
				"Activity (a) keeps the txnid.", r.Txnid),
			[]string{"OK", btnCopyTx}, func(label string) {
				toMenu()
				if label == btnCopyTx {
					sh.Copy(r.Txnid, "txnid")
				}
			})

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
				req := flow.Request()
				wei, _ := api.ParseWei(req.Amount)
				logAttempt(sh, flow, next, req.Source, req.Dest, wei, true)
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

// outcomeLabel is an outcome's name in the activity log.
func outcomeLabel(o Outcome) string {
	switch o {
	case Sent:
		return "sent"
	case NoResponse:
		return "no response"
	case AlreadyProcessed:
		return "already processed"
	case NodeError:
		return "node error"
	case SignatureRejected:
		return "signature rejected"
	case Rejected:
		return "rejected"
	}
	return "failed"
}

// logAttempt records one send attempt in the activity: what was sent where, its request
// id, outcome and txnid. c copies the txnid, or the request id if there's none.
func logAttempt(sh *ui.Shell, flow *Flow, r Result, source, dest string, wei *big.Int, retry bool) {
	what := "Send: " + outcomeLabel(r.Outcome)
	if retry {
		what = "Retry: " + outcomeLabel(r.Outcome)
	}
	eth := "?"
	if wei != nil {
		eth = api.FormatEther(wei) + " ETH"
	}
	summary := eth + " to " + state.ShortAddress(dest)
	details := []string{
		"From       " + source,
		"To         " + dest,
		"Amount     " + eth,
	}
	if wei != nil {
		details[2] += " (" + wei.String() + " wei)"
	}
	e := ui.Entry{What: what}
	if id := flow.Request().RequestID; id != "" {
		details = append(details, "Request id "+id)
		e.Copy, e.CopyIs = id, "request id"
	}
	if r.Txnid != "" {
		details = append(details, "Txnid      "+r.Txnid)
		summary += "  tx " + r.Txnid
		e.Copy, e.CopyIs = r.Txnid, "txnid"
	}
	if r.Err != nil {
		details = append(details, "Error      "+errText(r.Err))
	}
	e.Summary, e.Details = summary, details
	sh.Log(e)
}

// errText is the server's text for an api.Error, or the error itself.
func errText(err error) string {
	var e *api.Error
	if errors.As(err, &e) && e.Text != "" {
		return e.Text
	}
	return err.Error()
}
