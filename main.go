package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func getTokenInputForm(pages *tview.Pages, app *tview.Application) *tview.Form {
	tokenInput := tview.NewInputField().SetLabel("Token value: ").SetFieldWidth(256).SetAcceptanceFunc(tview.InputFieldMaxLength(256))
	form := tview.NewForm().
		AddFormItem(tokenInput).
		AddButton("Save", func() {
			token = tokenInput.GetText()
			pages.SwitchToPage("Menu")
		}).
		AddButton("Quit", func() {
			app.Stop()
		})
	form.SetBorder(true).SetTitle("Initialize application with JWT token").SetTitleAlign(tview.AlignLeft)
	return form
}

// Global state -- TODO: go idiom for global state
var token string

func getMainList(pages *tview.Pages) *tview.List {
	menuList := tview.NewList().
		AddItem("Generate key", "Generate a key for signing API requests", 'k', nil).
		AddItem("Register key", "Register API signing key", 'r', nil)

	menuList.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Rune() == 107 {
			pages.SwitchToPage("Keygen")
		} else if event.Rune() == 114 {
			pages.SwitchToPage("Register")
		}
		return event
	})

	return menuList
}

func main() {

	app := tview.NewApplication()

	var pages = tview.NewPages()

	list := getMainList(pages) //.SetBorder(true).SetTitle("Main list").SetTitleAlign(tview.AlignLeft)

	keygenTextView := tview.NewTextView().SetText("key generation")
	registerTextView := tview.NewTextView().SetText("key registration")

	pages.AddPage("Menu", list, true, true)
	pages.AddPage("Add Token", getTokenInputForm(pages, app), true, true)
	pages.AddPage("Keygen", keygenTextView, true, false)
	pages.AddPage("Register", registerTextView, true, false)

	if err := app.SetRoot(pages, true).SetFocus(pages).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
