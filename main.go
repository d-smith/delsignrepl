package main

import (
	"bytes"
	"delsignrepl/api"
	"delsignrepl/keys"
	"delsignrepl/state"
	"delsignrepl/token"
	"encoding/json"
	"fmt"
	"net/http"

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

func intsToStrings(ints []int) []string {
	strings := make([]string, len(ints))
	for i, v := range ints {
		strings[i] = fmt.Sprintf("%d", v)
	}
	return strings
}

func getWallets() ([]int, error) {
	return []int{1, 2, 3}, nil
}

func genAddressForm(pages *tview.Pages) {
	wallets, _ := getWallets()
	selection := 0
	form := tview.NewForm().
		AddDropDown("Select an option (hit Enter): ", intsToStrings(wallets), 0,
			func(val string, idx int) {
				selection = idx
			}).
		AddButton("Save", func() {
			modal := tview.NewModal().
				SetText(fmt.Sprintf("New address for wallet %d is 0x001",
					wallets[selection])).
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					if buttonLabel == "OK" {
						pages.SwitchToPage("Menu")
					}
				})

			pages.AddPage("NewAddress", modal, true, false)
			pages.SwitchToPage("NewAddress")
		}).
		AddButton("Cancel", func() {
			pages.SwitchToPage("Menu")
		})
	form.SetBorder(true).SetTitle("Select wallet for address gen").SetTitleAlign(tview.AlignLeft)
	pages.AddPage("Address Gen", form, true, false)
	pages.SwitchToPage("Address Gen")
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

func postWalletRequest() (int, error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:3010/api/v1/wallets", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)

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

func doGenerateWallet(pages *tview.Pages) {
	walletGenView := createWalletGenTextView(pages)
	walletGenView.Write([]byte("\nGenerating wallet...\n"))
	id, err := postWalletRequest()
	if err != nil {
		walletGenView.Write([]byte("Error: " + err.Error() + "\n"))
	} else {
		walletGenView.Write([]byte(fmt.Sprintf("Wallet ID: %d\n", id)))
	}

	walletGenView.Write([]byte("\nPress m to return to the main menu\n"))

	pages.AddPage("WalletGen", walletGenView, true, false)
	pages.SwitchToPage("WalletGen")
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

		keyReg := api.KeyReg{Email: email.(string),
			PubKey:                   state.PublicKeyDER,
			SignatureForRegistration: sig,
		}

		err = postKeyReg(keyReg)
		if err != nil {
			keyGenView.Write([]byte("Error: " + err.Error() + "\n"))
		} else {

			keyGenView.Write([]byte("Registered.\n"))
		}
	}

	keyGenView.Write([]byte("\nPress m to return to the main menu\n"))

	pages.AddPage("Keygen", keyGenView, true, false)
	pages.SwitchToPage("Keygen")
}

func getMainList(pages *tview.Pages, app *tview.Application) *tview.List {
	menuList := tview.NewList().
		AddItem("Generate key", "Generate a key for signing API requests", 'k', nil).
		AddItem("Register key", "Register API signing key", 'r', nil).
		AddItem("Create wallet", "Create a wallet", 'w', nil).
		AddItem("Generate address", "Generate an address for a wallet", 'a', nil).
		AddItem("Quit", "Exit", 'q', nil)

	menuList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 107 {
			doKeyGeneration(pages)
		} else if event.Rune() == 114 {
			doKeyRegistration(pages)
		} else if event.Rune() == 113 {
			app.Stop()
		} else if event.Rune() == 119 {
			doGenerateWallet(pages)
		} else if event.Rune() == 97 {
			genAddressForm(pages)
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
