package ui_test

import (
	"delsignrepl/api"
	"delsignrepl/ui"
	"delsignrepl/ui/uitest"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// testShell has a Home page, an action 'o' that opens screen "One" (an input field) and
// an action 'p' that opens screen "Two" (a text view).
func testShell(t *testing.T) (*ui.Shell, *uitest.Sim, *int) {
	t.Helper()
	app := tview.NewApplication()
	sh := ui.NewShell(app)
	opened := 0
	sh.AddAction("Open one", 'o', func() {
		opened++
		sh.Show("One", tview.NewInputField().SetLabel("One field: "))
	})
	sh.AddAction("Open two", 'p', func() {
		sh.Show("Two", tview.NewTextView().SetText("two text"))
	})
	sh.SetHome(tview.NewTextView().SetText("home text"))
	return sh, uitest.Start(t, app, 100, 24, sh.Run), &opened
}

func TestShellLayoutAndFocus(t *testing.T) {
	sh, sim, opened := testShell(t)
	sim.Shows("delsign", "Not logged in", "Open one", "home text", "Esc back", "F2 mouse:on")
	if sim.App.GetFocus() != sh.Sidebar {
		t.Fatal("Home should leave the focus on the sidebar")
	}

	sim.Type("o")
	sim.Shows("One field:")
	if *opened != 1 {
		t.Fatalf("shortcut ran the action %d times", *opened)
	}
	if _, ok := sim.App.GetFocus().(*tview.InputField); !ok {
		t.Fatalf("opening a screen should focus it, focus is %T", sim.App.GetFocus())
	}

	sim.Type("oq?")
	if *opened != 1 || sh.Current() != "One" {
		t.Error("typing in a field triggered a shortcut or help")
	}

	sim.Do(func() { sh.SetHeader("you@example.com"); sh.Status("saved") })
	sim.Shows("you@example.com", "saved")
}

func TestShellEscGoesBack(t *testing.T) {
	sh, sim, _ := testShell(t)
	sim.Type("p") // Home -> Two
	sim.Do(func() { sh.Show("One", tview.NewInputField()) })
	if sh.Current() != "One" {
		t.Fatalf("on %q", sh.Current())
	}

	sim.Key(tcell.KeyEscape) // from an input field too
	if sh.Current() != "Two" {
		t.Fatalf("Esc from One went to %q, want Two", sh.Current())
	}
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage || sim.App.GetFocus() != sh.Sidebar {
		t.Fatalf("Esc from Two went to %q", sh.Current())
	}
	sim.Key(tcell.KeyEscape) // Home stays Home
	if sh.Current() != ui.HomePage {
		t.Fatalf("Esc on Home went to %q", sh.Current())
	}
}

func TestShellHistoryDoesntGrowInCircles(t *testing.T) {
	sh, sim, _ := testShell(t)
	sim.Do(func() {
		for i := 0; i < 5; i++ {
			sh.Show("One", nil)
			sh.Show("Two", tview.NewTextView())
			sh.Show("One", tview.NewInputField())
		}
	})
	sim.Key(tcell.KeyEscape)
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage {
		t.Errorf("two Escs from a loop should reach Home, on %q", sh.Current())
	}
}

func TestShellModalReturnsToScreenUnderIt(t *testing.T) {
	sh, sim, _ := testShell(t)
	sim.Type("p")
	var got []string
	sim.Do(func() {
		sh.Modal("Sure?", []string{"Yes", "No"}, func(l string) { got = append(got, l) })
	})
	sim.Shows("Sure?", "two text")
	if sh.Current() != "Two" {
		t.Errorf("Current under a modal = %q, want Two", sh.Current())
	}

	sim.Key(tcell.KeyEnter) // Yes
	if len(got) != 1 || got[0] != "Yes" || sh.Current() != "Two" {
		t.Fatalf("after Yes: labels %v, on %q", got, sh.Current())
	}
	if _, ok := sim.App.GetFocus().(*tview.TextView); !ok {
		t.Errorf("focus should return to the screen, is %T", sim.App.GetFocus())
	}

	sim.Do(func() { sh.Message("Note") })
	sim.Key(tcell.KeyEscape) // closes the modal, doesn't leave the screen
	if sh.Current() != "Two" || strings.Contains(sim.Text(), "Note") {
		t.Errorf("Esc on a modal: on %q, screen:\n%s", sh.Current(), sim.Text())
	}

	// A modal on Home gives the focus back to the sidebar.
	sim.Do(func() { sh.Home(); sh.Message("hi") })
	sim.Key(tcell.KeyEnter)
	if sim.App.GetFocus() != sh.Sidebar {
		t.Errorf("after a modal on Home, focus is %T", sim.App.GetFocus())
	}
}

func TestShellHelpAndMouseKeys(t *testing.T) {
	sh, sim, _ := testShell(t)
	sim.Type("?")
	sim.Shows("Esc               back")
	if sh.Current() != ui.HelpPage {
		t.Fatalf("? opened %q", sh.Current())
	}
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage {
		t.Errorf("Esc from help went to %q", sh.Current())
	}

	sim.Key(tcell.KeyF2)
	if sh.Mouse() {
		t.Error("F2 didn't turn the mouse off")
	}
	sim.Shows("F2 mouse:off")
	sim.Key(tcell.KeyF2)
	if !sh.Mouse() {
		t.Error("F2 didn't turn the mouse back on")
	}
}

func TestShellErrorRoutesTokenProblemsToLogin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Invalid token", 403) // /authping agrees: the token is the problem
	}))
	defer srv.Close()
	t.Setenv("DELSIGN_API_URL", srv.URL)

	sh, sim, _ := testShell(t)
	sim.Do(func() { sh.Content.AddPage(ui.LoginPage, tview.NewTextView().SetText("login page"), true, false) })
	sim.Type("p")
	sim.Do(func() { sh.Error("Error listing wallets", &api.Error{Status: 403, Text: "Forbidden"}) })
	sim.Shows("Error listing wallets", "Log in again")
	sim.Key(tcell.KeyEnter)
	if sh.Current() != ui.LoginPage {
		t.Errorf("OK on a token error went to %q, want the login page", sh.Current())
	}

	sim.Do(func() { sh.Show("Two", nil); sh.Error("Oops", &api.Error{Status: 404, Text: "nope"}) })
	sim.Key(tcell.KeyEnter)
	if sh.Current() != "Two" {
		t.Errorf("OK on a non-token error went to %q, want the screen under it", sh.Current())
	}
}

// Esc closes an open dropdown first, and only the next Esc leaves the screen.
func TestShellEscClosesDropdownFirst(t *testing.T) {
	sh, sim, _ := testShell(t)
	sim.Do(func() {
		sh.Show("Form", tview.NewForm().AddDropDown("Pick: ", []string{"apple", "banana"}, 0, nil))
	})
	sim.Key(tcell.KeyEnter) // open the dropdown
	sim.Shows("banana")
	sim.Key(tcell.KeyEscape)
	if sh.Current() != "Form" {
		t.Fatalf("Esc with the dropdown open left the screen, on %q", sh.Current())
	}
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.HomePage {
		t.Errorf("second Esc went to %q", sh.Current())
	}
}

// A TrimPaste field trims pastes, and the shell still sees it as a text field, so ? is
// typed rather than opening help.
func TestTrimPasteField(t *testing.T) {
	sh, sim, _ := testShell(t)
	digits := func(text string, _ rune) bool { return strings.Trim(text, "0123456789") == "" }
	field := tview.NewInputField().SetLabel("Amount: ").SetAcceptanceFunc(digits)
	sim.Do(func() { sh.Show("Form", tview.NewForm().AddFormItem(ui.TrimPaste(field))) })

	// A real bracketed paste of a copied line, trailing newline included.
	sim.Paste(" 1000\n", func() bool { return field.GetText() != "" })
	if field.GetText() != "1000" || sh.Current() != "Form" {
		t.Errorf("pasted %q (on %q), want 1000", field.GetText(), sh.Current())
	}

	other := tview.NewInputField().SetLabel("Note: ")
	sim.Do(func() { sh.Show("Note", tview.NewForm().AddFormItem(ui.TrimPaste(other))) })
	sim.Type("why?")
	if sh.Current() != "Note" || other.GetText() != "why?" {
		t.Errorf("? in a wrapped field: on %q, text %q", sh.Current(), other.GetText())
	}
}
