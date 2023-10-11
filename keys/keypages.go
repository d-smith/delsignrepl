package keys

import (
	"bytes"
	"delsignrepl/api"
	"delsignrepl/state"
	"delsignrepl/token"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func DoKeyGeneration(pages *tview.Pages) {

	priv, pub := Generate()
	privEnc, pubEnc := Encode(priv, pub)

	state.PrivateKey = priv
	state.PublicKeyDER = pubEnc

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Keys generated\nPrivate key: %s\nPublic key: %s",
			privEnc, pubEnc)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				pages.SwitchToPage("Menu")
			}
		})

	pages.AddPage("Keygen", modal, true, false)
	pages.SwitchToPage("Keygen")
}

func doKeyRegErrorModel(pages *tview.Pages, err error) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Error registering key: %s", err.Error())).
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

func DoKeyRegistration(pages *tview.Pages) {

	email, err := token.ExtractEmailFromToken(state.Token, "secret")
	if err != nil {
		doKeyRegErrorModel(pages, err)
	} else {

		sig := Sign(email.(string), state.PrivateKey)

		keyReg := api.KeyReg{Email: email.(string),
			PubKey:                   state.PublicKeyDER,
			SignatureForRegistration: sig,
		}

		err = postKeyReg(keyReg)
		if err != nil {
			doKeyRegErrorModel(pages, err)
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

func postKeyReg(keyReg api.KeyReg) error {

	keyRegJSON, err := json.Marshal(keyReg)
	if err != nil {
		return err
	}
	bodyReader := bytes.NewReader(keyRegJSON)
	req, err := http.NewRequest(http.MethodPost, "http://localhost:3010/api/v1/keyreg", bodyReader)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("error registering key")
	}

	return nil
}
