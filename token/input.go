package token

import (
	"delsignrepl/state"

	"github.com/rivo/tview"
)

func GetTokenInputForm(pages *tview.Pages, app *tview.Application) *tview.Form {
	tokenInput := tview.NewInputField().SetLabel("Token value: ").SetFieldWidth(256).SetAcceptanceFunc(tview.InputFieldMaxLength(256))
	form := tview.NewForm().
		AddFormItem(tokenInput).
		AddCheckbox("Fake it", false, func(checked bool) {
			if checked {
				tokenInput.SetText("fake")
			} else {
				tokenInput.SetText("")
			}
		}).
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
