package main

import (
	"bytes"
	"delsignrepl/api"
	"delsignrepl/keys"
	"delsignrepl/state"
	"delsignrepl/token"
	"encoding/json"
	"fmt"
	"math/big"
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
	var wallets []int
	req, err := http.NewRequest(http.MethodGet, "http://localhost:3010/api/v1/wallets", nil)
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return wallets, err
	}

	if resp.StatusCode != 201 {
		return wallets, fmt.Errorf("error generating wallet")
	}

	err = json.NewDecoder(resp.Body).Decode(&wallets)
	if err != nil {
		return wallets, err
	}

	return wallets, nil
}

func doAddressGeneration(pages *tview.Pages) {
	wallets, _ := getWallets()
	selection := 0
	form := tview.NewForm().
		AddDropDown("Select an option (hit Enter): ", intsToStrings(wallets), 0,
			func(val string, idx int) {
				selection = idx
			}).
		AddButton("Save", func() {
			var modal *tview.Modal
			address, err := createWalletAddress(wallets[selection])
			if err != nil {
				modal = tview.NewModal().
					SetText(fmt.Sprintf("Error creating address for wallet: %s",
						err.Error())).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonLabel == "OK" {
							pages.SwitchToPage("Menu")
						}
					})
			} else {

				modal = tview.NewModal().
					SetText(fmt.Sprintf("New address for wallet %d is %s",
						wallets[selection], address)).
					AddButtons([]string{"OK"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonLabel == "OK" {
							pages.SwitchToPage("Menu")
						}
					})
			}

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

func createWalletAddress(walletId int) (string, error) {
	req, err := http.NewRequest(http.MethodPost, "http://localhost:3010/api/v1/wallets/"+fmt.Sprintf("%d", walletId)+"/addresses", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != 201 {
		return "", fmt.Errorf("error generating wallet address")
	}

	var eoa struct {
		Address string `json:"eoa"`
	}

	err = json.NewDecoder(resp.Body).Decode(&eoa)
	if err != nil {
		return "", err
	}

	return eoa.Address, nil
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

type WalletAddressPair struct {
	WalletId int    `json:"walletId"`
	Address  string `json:"address"`
}

func getWalletsAndAddresses() ([]WalletAddressPair, error) {
	var walletAddressPairs []WalletAddressPair
	req, err := http.NewRequest(http.MethodGet, "http://localhost:3010/api/v1/walletctx", nil)
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return walletAddressPairs, err
	}

	if resp.StatusCode != 200 {
		return walletAddressPairs, fmt.Errorf("error retrieving wallets and addresses for user")
	}

	err = json.NewDecoder(resp.Body).Decode(&walletAddressPairs)
	if err != nil {
		return walletAddressPairs, err
	}

	return walletAddressPairs, nil
}

func pairsToStrings(pairs []WalletAddressPair) []string {
	strings := make([]string, len(pairs))
	for i, v := range pairs {
		strings[i] = fmt.Sprintf("Wallet %6d | %s", v.WalletId, v.Address)
	}
	return strings
}

func getBalanceForAddress(address string) (*big.Int, error) {

	req, err := http.NewRequest(http.MethodGet,
		"http://localhost:3010/api/v1/wallets/balance/"+address, nil)
	req.Header.Set("Authorization", "Bearer "+state.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error retrieving wallets and addresses for user")
	}

	var accountBalance struct {
		Address string   `json:"address"`
		Amount  *big.Int `json:"amount"`
	}

	err = json.NewDecoder(resp.Body).Decode(&accountBalance)
	if err != nil {
		return nil, err
	}

	return accountBalance.Amount, nil
}

func doGetBalance(pages *tview.Pages) {
	var msg string
	if state.WalletId == 0 || state.Address == "" {
		msg = "Wallet and address context not set"
	} else {

		balance, err := getBalanceForAddress(state.Address)
		if err != nil {
			msg = "Error retrieving balance: " + err.Error()
		} else {
			msg = fmt.Sprintf("Balance for wallet %d, address %s is %d",
				state.WalletId, state.Address, balance)
		}

	}

	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "OK" {
				pages.SwitchToPage("Menu")
			}

		})

	pages.AddPage("Balance", modal, true, false)
	pages.SwitchToPage("Balance")

}

func doSetWalletAndAccountCtx(pages *tview.Pages) {
	selection := 0
	walletAddressPairs, err := getWalletsAndAddresses()
	form := tview.NewForm().
		AddButton("Done", func() {
			state.WalletId = walletAddressPairs[selection].WalletId
			state.Address = walletAddressPairs[selection].Address
			pages.SwitchToPage("Menu")
		})

	if err == nil {

		form.
			AddDropDown("Select an option (hit Enter): ", pairsToStrings(walletAddressPairs), 0,
				func(val string, idx int) {
					selection = idx
				}).
			AddTextView("Status", fmt.Sprintf("%d items to dosplay", len(walletAddressPairs)), 50, 1, true, true)
	} else {
		form.AddTextView("Status", err.Error(), 50, 1, true, true)

	}

	form.SetBorder(true).SetTitle("Select wallet and address for txn context").SetTitleAlign(tview.AlignLeft)
	pages.AddPage("SetCtx", form, true, false)
	pages.SwitchToPage("SetCtx")
}

func doWalletGeneration(pages *tview.Pages) {
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
		AddItem("Set wallet/address context", "Set wallet/address context", 'c', nil).
		AddItem("Get balance", "Get balance for current wallet", 'b', nil).
		AddItem("Quit", "Exit", 'q', nil)

	menuList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 'k' {
			doKeyGeneration(pages)
		} else if event.Rune() == 'r' {
			doKeyRegistration(pages)
		} else if event.Rune() == 'q' {
			app.Stop()
		} else if event.Rune() == 'w' {
			doWalletGeneration(pages)
		} else if event.Rune() == 'a' {
			doAddressGeneration(pages)
		} else if event.Rune() == 'c' {
			doSetWalletAndAccountCtx(pages)
		} else if event.Rune() == 'b' {
			doGetBalance(pages)
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
