package keys

import (
	"delsignrepl/state"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func DoKeyGeneration(pages *tview.Pages) {
	keyGenView := createKeyGenTextView(pages)
	keyGenView.Write([]byte("\nGenerating your key...\n"))
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

func DoKeyGeneration1(pages *tview.Pages) {
	keyGenView := createKeyGenTextView(pages)
	keyGenView.Write([]byte("\nGenerating your key...\n"))
	priv, pub := Generate()
	privEnc, pubEnc := Encode(priv, pub)

	state.PrivateKey = priv
	state.PublicKeyDER = pubEnc

	keyGenView.Write([]byte("Private key: " + privEnc + "\n"))
	keyGenView.Write([]byte("Public key: " + pubEnc + "\n"))
	keyGenView.Write([]byte("\nPress m to return to the main menu\n"))

	pages.AddPage("Keygen", keyGenView, true, false)
	pages.SwitchToPage("Keygen")
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
