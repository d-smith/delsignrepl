package main

import (
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

func getMainList() *tview.List {
	return tview.NewList().
		AddItem("List item 1", "Some explanatory text", 'a', nil).
		AddItem("List item 2", "Some explanatory text", 'b', nil).
		AddItem("List item 3", "Some explanatory text", 'c', nil).
		AddItem("List item 4", "Some explanatory text", 'd', nil)
}

func main() {

	app := tview.NewApplication()

	var pages = tview.NewPages()

	list := getMainList() //.SetBorder(true).SetTitle("Main list").SetTitleAlign(tview.AlignLeft)

	pages.AddPage("Menu", list, true, true)
	pages.AddPage("Add Token", getTokenInputForm(pages, app), true, true)

	if err := app.SetRoot(pages, true).SetFocus(pages).EnableMouse(true).Run(); err != nil {
		panic(err)
	}

}
