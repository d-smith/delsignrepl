package main

import (
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// The real layout starts on the login page, and the old actions run from the sidebar
// with their screens in the content area.
func TestAppStartsOnLoginInsideShell(t *testing.T) {
	t.Setenv("DELSIGN_DEV_JWT_SECRET", "")
	app := tview.NewApplication()
	sh := buildShell(app)
	sim := uitest.Start(t, app, 120, 30, sh.Run)

	sim.Shows("Log in with a JWT", "Not logged in", "Generate key", "Send ETH", "Quit")
	if app.GetFocus() == sh.Sidebar {
		t.Error("the login page should have the focus at start")
	}

	sim.Do(func() { sh.Content.SwitchToPage(ui.HomePage) })
	sim.Type("i") // Key status, before logging in
	sim.Shows("Log in first")
	sim.Key(tcell.KeyEnter) // OK
	sim.Shows("Choose an action")
	if app.GetFocus() != sh.Sidebar {
		t.Error("OK on the modal should return to Home and the sidebar")
	}
}
