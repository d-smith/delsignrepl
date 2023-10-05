package main

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"delsignrepl/keys"
	"delsignrepl/state"
	"delsignrepl/token"
	"encoding/hex"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func getTokenInputForm(pages *tview.Pages, app *tview.Application) *tview.Form {
	tokenInput := tview.NewInputField().SetLabel("Token value: ").SetFieldWidth(256).SetAcceptanceFunc(tview.InputFieldMaxLength(256))
	form := tview.NewForm().
		AddFormItem(tokenInput).
		AddButton("Save", func() {
			state.Token = tokenInput.GetText()
			pages.SwitchToPage("Menu")
		}).
		AddButton("Quit", func() {
			app.Stop()
		})
	form.SetBorder(true).SetTitle("Initialize application with JWT token").SetTitleAlign(tview.AlignLeft)
	return form
}

func doKeyGeneration(pages *tview.Pages) {
	keyGenView := createKeyGenTextView(pages)
	keyGenView.Write([]byte("\nGenerating your key...\n"))
	priv, pub := keys.Generate()
	privEnc, pubEnc := keys.Encode(priv, pub)

	state.PrivateKey = priv
	state.PublicKeyDER = pubEnc

	keyGenView.Write([]byte("Private key: " + privEnc + "\n"))
	keyGenView.Write([]byte("Public key: " + pubEnc + "\n"))
	keyGenView.Write([]byte("\nPress m to return to the main menu\n"))

	pages.AddPage("Keygen", keyGenView, true, false)
	pages.SwitchToPage("Keygen")
}

func doKeyRegistration(pages *tview.Pages) {
	keyGenView := createRegisterTextView(pages)
	keyGenView.Write([]byte("\nRegistering your key...\n"))
	keyGenView.Write([]byte("PubKey: " + state.PublicKeyDER + "\n"))

	email, err := token.ExtractEmailFromToken(state.Token, "secret")
	if err != nil {
		keyGenView.Write([]byte("Error: " + err.Error() + "\n"))
	} else {

		keyGenView.Write([]byte("Email: " + email.(string) + "\n"))
		sig := keys.Sign(email.(string), state.PrivateKey)
		keyGenView.Write([]byte("Signature: " + sig + "\n"))

		hash := sha256.Sum256([]byte(email.(string)))
		decodedSig, _ := hex.DecodeString(sig)
		valid := ecdsa.VerifyASN1(&state.PrivateKey.PublicKey, hash[:], decodedSig)
		if valid {
			keyGenView.Write([]byte("Signature verified\n"))
		} else {
			keyGenView.Write([]byte("Signature not verified\n"))
		}

		keyGenView.Write([]byte("Registered.\n"))
	}

	keyGenView.Write([]byte("\nPress m to return to the main menu\n"))

	pages.AddPage("Keygen", keyGenView, true, false)
	pages.SwitchToPage("Keygen")
}

func getMainList(pages *tview.Pages, app *tview.Application) *tview.List {
	menuList := tview.NewList().
		AddItem("Generate key", "Generate a key for signing API requests", 'k', nil).
		AddItem("Register key", "Register API signing key", 'r', nil).
		AddItem("Quit", "Exit", 'q', nil)

	menuList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 107 {
			doKeyGeneration(pages)
		} else if event.Rune() == 114 {
			doKeyRegistration(pages)
		} else if event.Rune() == 113 {
			app.Stop()
		}
		return event
	})

	return menuList
}

func createKeyGenTextView(pages *tview.Pages) *tview.TextView {
	textView := tview.NewTextView().SetText("Key generation")
	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 109 {
			pages.SwitchToPage("Menu")
		}
		return event
	})

	return textView
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

func main() {

	app := tview.NewApplication()

	var pages = tview.NewPages()

	list := getMainList(pages, app) //.SetBorder(true).SetTitle("Main list").SetTitleAlign(tview.AlignLeft)

	pages.AddPage("Menu", list, true, true)
	pages.AddPage("Add Token", getTokenInputForm(pages, app), true, true)
	//pages.AddPage("Keygen", createKeyGenTextView(pages), true, false)
	//pages.AddPage("Register", createRegisterTextView(pages), true, false)

	if err := app.SetRoot(pages, true).SetFocus(pages).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
