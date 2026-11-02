package keys

import (
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/token"
	"delsignrepl/ui"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func DoKeyGeneration(pages *tview.Pages) {

	priv, pub := Generate()
	_, pubEnc := Encode(priv, pub)

	state.PrivateKey = priv
	state.PublicKeyDER = pubEnc

	modal := tview.NewModal().
		SetText(fmt.Sprintf("RSA user key pair generated.")).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				pages.SwitchToPage("Menu")
			}
		})

	pages.AddPage("Keygen", modal, true, false)
	pages.SwitchToPage("Keygen")
}

func doKeyRegErrorModel(pages *tview.Pages, err error, appToken string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Error registering key: %s", ui.Describe(err, appToken))).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				pages.SwitchToPage("Menu")
			}
		})

	pages.AddPage("KeyRegError", modal, true, false)
	pages.SwitchToPage("KeyRegError")
}

func doKeyRegOKModel(pages *tview.Pages) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Key registered")).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				pages.SwitchToPage("Menu")
			}
		})

	pages.AddPage("KeyRegError", modal, true, false)
	pages.SwitchToPage("KeyRegError")
}

func DoKeyRegistration(pages *tview.Pages, appToken string) {

	email, err := token.ExtractEmailFromToken(appToken, "secret")
	if err != nil {
		doKeyRegErrorModel(pages, err, appToken)
	} else {

		sig, err := Sign(email.(string), state.PrivateKey)
		if err != nil {
			doKeyRegErrorModel(pages, err, appToken)
			return
		}

		keyReg := api.KeyReg{Email: email.(string),
			PubKey:                   state.PublicKeyDER,
			SignatureForRegistration: sig,
		}

		err = api.New(appToken).RegisterKey(keyReg)
		if err != nil {
			doKeyRegErrorModel(pages, err, appToken)
		} else {

			doKeyRegOKModel(pages)
		}
	}
}

func createRegisterTextView(pages *tview.Pages) *tview.TextView {
	textView := tview.NewTextView().SetText("Key registration")
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 109 {
			pages.SwitchToPage("Menu")
		}
		return event
	})

	return textView
}
