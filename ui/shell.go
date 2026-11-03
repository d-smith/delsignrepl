package ui

import "github.com/rivo/tview"

// HomePage is the content page shown when no screen is open. Its name is "Menu" because
// the existing screens return there by name; Phase 1 of plan/03 replaces the named
// returns with Shell.Back.
const HomePage = "Menu"

// FooterHints are the key hints always shown in the footer.
const FooterHints = "Tab/arrows move  Enter select  q quit (from the menu)"

// Shell is the app's fixed layout (plan/03, "Target layout"):
//
//	header  (who, key status, wallet context)
//	sidebar | content
//	footer  (key hints, last status message)
//
// Screens are pages in Content. The sidebar holds the actions.
type Shell struct {
	App     *tview.Application
	Root    *tview.Flex
	Header  *tview.TextView
	Sidebar *tview.List
	Content *tview.Pages
	Footer  *tview.TextView

	status string
}

// NewShell builds the layout. Add actions with AddAction and screens to Content.
func NewShell(app *tview.Application) *Shell {
	s := &Shell{
		App:     app,
		Header:  tview.NewTextView().SetDynamicColors(true),
		Sidebar: tview.NewList().ShowSecondaryText(false).SetHighlightFullLine(true),
		Content: tview.NewPages(),
		Footer:  tview.NewTextView().SetDynamicColors(true),
	}
	s.Header.SetBorder(true).SetTitle(" delsign ").SetTitleAlign(tview.AlignLeft)
	s.Sidebar.SetBorder(true).SetTitle(" Menu ").SetTitleAlign(tview.AlignLeft)

	body := tview.NewFlex().
		AddItem(s.Sidebar, 28, 0, true).
		AddItem(s.Content, 0, 1, false)
	s.Root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(s.Header, 3, 0, false).
		AddItem(body, 0, 1, true).
		AddItem(s.Footer, 1, 0, false)

	// Screens switch pages by name. Follow them with the focus: Home gives it back to
	// the sidebar, anything else takes it.
	s.Content.SetChangedFunc(s.focusFront)

	s.SetHeader("Not logged in")
	s.drawFooter()
	return s
}

// AddAction adds a sidebar item. key is its shortcut ('q' quits by convention).
func (s *Shell) AddAction(label string, key rune, do func()) *Shell {
	s.Sidebar.AddItem(label, "", key, do)
	return s
}

// SetHeader replaces the header text.
func (s *Shell) SetHeader(text string) {
	s.Header.SetText(text)
}

// Status shows msg in the footer after the key hints, until the next Status.
func (s *Shell) Status(msg string) {
	s.status = msg
	s.drawFooter()
}

func (s *Shell) drawFooter() {
	text := "[::d]" + FooterHints + "[::-]"
	if s.status != "" {
		text += "  |  " + tview.Escape(s.status)
	}
	s.Footer.SetText(text)
}

// Run starts the app with the shell as root. Mouse and bracketed paste are on.
func (s *Shell) Run() error {
	s.App.SetRoot(s.Root, true).EnableMouse(true).EnablePaste(true)
	s.focusFront() // SetRoot moved the focus to the root
	return s.App.Run()
}

// focusFront gives the focus to the sidebar on Home, or to the open screen.
func (s *Shell) focusFront() {
	if name, _ := s.Content.GetFrontPage(); name == HomePage || name == "" {
		s.App.SetFocus(s.Sidebar)
	} else {
		s.App.SetFocus(s.Content)
	}
}
