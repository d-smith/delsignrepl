package ui_test

import (
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"testing"

	"github.com/rivo/tview"
)

func TestShellLayoutAndFocus(t *testing.T) {
	app := tview.NewApplication()
	sh := ui.NewShell(app)
	opened := 0
	sh.AddAction("Open thing", 'o', func() {
		opened++
		sh.Content.AddAndSwitchToPage("Thing", tview.NewInputField().SetLabel("Thing field: "), true)
	})
	sh.Content.AddPage(ui.HomePage, tview.NewTextView().SetText("home text"), true, true)
	sim := uitest.Start(t, app, 100, 20, sh.Run)

	sim.Shows("delsign", "Not logged in", "Open thing", "home text", ui.FooterHints)
	if app.GetFocus() != sh.Sidebar {
		t.Fatal("Home should leave the focus on the sidebar")
	}

	sim.Type("o")
	sim.Shows("Thing field:")
	if opened != 1 {
		t.Fatalf("shortcut ran the action %d times", opened)
	}
	if _, ok := app.GetFocus().(*tview.InputField); !ok {
		t.Fatalf("opening a screen should focus it, focus is %T", app.GetFocus())
	}

	sim.Type("o")
	if opened != 1 {
		t.Error("typing in the screen triggered a sidebar shortcut")
	}

	sim.Do(func() { sh.Content.SwitchToPage(ui.HomePage) })
	if app.GetFocus() != sh.Sidebar {
		t.Error("returning Home should give the focus back to the sidebar")
	}

	sim.Do(func() { sh.SetHeader("you@example.com"); sh.Status("saved") })
	sim.Shows("you@example.com", "saved")
}
