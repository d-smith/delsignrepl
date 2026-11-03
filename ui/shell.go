package ui

import (
	"delsignrepl/api"
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// HomePage is the content page shown when no screen is open.
const HomePage = "Home"

// HelpPage is the key help screen ('?').
const HelpPage = "Help"

const modalPrefix = "modal:"

// Shell is the app's fixed layout (plan/03, "Target layout"):
//
//	header  (who, key status, wallet context)
//	sidebar | content
//	footer  (key hints, last status message)
//
// Screens are pages in Content, opened with Show and left with Back or Home. Modals
// overlay the current screen and return to it when closed.
type Shell struct {
	App     *tview.Application
	Root    *tview.Flex
	Header  *tview.TextView
	Sidebar *tview.List
	Content *tview.Pages
	Footer  *tview.TextView

	// Token returns the current API token, for the 403 check in Error. Nil means none.
	Token func() string

	history []string // screens to go Back to, oldest first
	modals  int      // counter for unique modal page names
	mouse   bool
	status  string
}

// NewShell builds the layout. Add actions with AddAction, Home with SetHome, and other
// screens with Show.
func NewShell(app *tview.Application) *Shell {
	s := &Shell{
		App:     app,
		Header:  tview.NewTextView().SetDynamicColors(true),
		Sidebar: tview.NewList().ShowSecondaryText(false).SetHighlightFullLine(true),
		Content: tview.NewPages(),
		Footer:  tview.NewTextView().SetDynamicColors(true),
		mouse:   true,
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

	// The focus follows the content: Home gives it to the sidebar, anything else
	// (a screen or a modal) takes it.
	s.Content.SetChangedFunc(s.focusFront)
	s.Content.AddPage(HelpPage, helpView(), true, false)
	app.SetInputCapture(s.globalKeys)

	s.SetHeader("Not logged in")
	s.drawFooter()
	return s
}

// AddAction adds a sidebar item. key is its shortcut, active while the sidebar has the
// focus ('q' quits by convention).
func (s *Shell) AddAction(label string, key rune, do func()) *Shell {
	s.Sidebar.AddItem(label, "", key, do)
	return s
}

// SetHome sets the Home screen's content.
func (s *Shell) SetHome(p tview.Primitive) {
	s.Content.AddPage(HomePage, p, true, s.front() == "" || s.front() == HomePage)
}

// Show opens screen name, replacing any page of that name if p is non-nil, or switching
// to an existing page if p is nil. The screen it leaves is remembered for Back.
func (s *Shell) Show(name string, p tview.Primitive) {
	s.closeModals()
	cur := s.front()
	if cur != "" && cur != name {
		s.remember(cur)
	}
	if p != nil {
		s.Content.AddAndSwitchToPage(name, p, true)
	} else {
		s.Content.SwitchToPage(name)
	}
}

// remember pushes name onto the history, cutting the history back if name is already in
// it so that going round in circles doesn't grow it.
func (s *Shell) remember(name string) {
	for i, h := range s.history {
		if h == name {
			s.history = s.history[:i]
			break
		}
	}
	s.history = append(s.history, name)
}

// Back returns to the previous screen, or Home if there is none.
func (s *Shell) Back() {
	s.closeModals()
	for len(s.history) > 0 {
		prev := s.history[len(s.history)-1]
		s.history = s.history[:len(s.history)-1]
		if s.Content.HasPage(prev) && prev != s.front() {
			s.Content.SwitchToPage(prev)
			return
		}
	}
	s.Home()
}

// Home closes everything and shows Home, forgetting the history.
func (s *Shell) Home() {
	s.closeModals()
	s.history = nil
	s.Content.SwitchToPage(HomePage)
}

// Current returns the name of the screen shown (under any modal).
func (s *Shell) Current() string {
	name := ""
	for _, n := range s.Content.GetPageNames(true) {
		if !strings.HasPrefix(n, modalPrefix) {
			name = n
		}
	}
	return name
}

// Modal shows text with buttons over the current screen. Choosing a button (or Esc,
// which gives label "") closes it, returns to the screen under it, then calls done (if
// non-nil) with the button's label.
func (s *Shell) Modal(text string, buttons []string, done func(label string)) {
	s.modals++
	name := fmt.Sprintf("%s%d", modalPrefix, s.modals)
	m := tview.NewModal().SetText(text).AddButtons(buttons)
	m.SetDoneFunc(func(_ int, label string) {
		s.Content.RemovePage(name)
		s.focusFront() // RemovePage re-focuses the content itself, even on Home
		if done != nil {
			done(label)
		}
	})
	s.Content.AddPage(name, m, true, true)
}

// Message shows text with an OK button over the current screen.
func (s *Shell) Message(text string) {
	s.Modal(text, []string{"OK"}, nil)
}

// Error shows prefix and the described error (see describeErr). If the backend confirms
// a token problem, OK opens the login page.
func (s *Shell) Error(prefix string, err error) {
	token := ""
	if s.Token != nil {
		token = s.Token()
	}
	msg, tokenProblem := describeErr(err, func() error { return api.New(token).AuthPing() })
	s.Modal(prefix+": "+msg, []string{"OK"}, func(string) {
		if tokenProblem {
			s.Show(LoginPage, nil)
		}
	})
}

func (s *Shell) closeModals() {
	for _, n := range s.Content.GetPageNames(false) {
		if strings.HasPrefix(n, modalPrefix) {
			s.Content.RemovePage(n)
		}
	}
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

// Mouse reports whether the app captures the mouse.
func (s *Shell) Mouse() bool { return s.mouse }

// SetMouse turns mouse capture on (click to select) or off (the terminal's own
// drag-to-select and copy work).
func (s *Shell) SetMouse(on bool) {
	s.mouse = on
	s.App.EnableMouse(on)
	s.drawFooter()
}

// Hints returns the footer's key hints.
func (s *Shell) Hints() string {
	mouse := "off"
	if s.mouse {
		mouse = "on"
	}
	return "Tab move  Enter select  Esc back  ? help  F2 mouse:" + mouse + "  q quit"
}

func (s *Shell) drawFooter() {
	text := "[::d]" + s.Hints() + "[::-]"
	if s.status != "" {
		text += "  |  " + tview.Escape(s.status)
	}
	s.Footer.SetText(text)
}

// Run starts the app with the shell as root. Bracketed paste is on.
func (s *Shell) Run() error {
	s.App.SetRoot(s.Root, true).EnableMouse(s.mouse).EnablePaste(true)
	s.focusFront() // SetRoot moved the focus to the root
	return s.App.Run()
}

func (s *Shell) front() string {
	name, _ := s.Content.GetFrontPage()
	return name
}

func (s *Shell) modalOpen() bool {
	return strings.HasPrefix(s.front(), modalPrefix)
}

// focusFront gives the focus to the sidebar on Home, or to the open screen or modal.
func (s *Shell) focusFront() {
	if name := s.front(); name == HomePage || name == "" {
		s.App.SetFocus(s.Sidebar)
	} else {
		s.App.SetFocus(s.Content)
	}
}

// typing reports whether the focus is somewhere keys are text: an input field, a text
// area, or an open dropdown.
func (s *Shell) typing() bool {
	switch s.App.GetFocus().(type) {
	case *tview.InputField, *tview.TextArea:
		return true
	}
	return s.dropdownOpen() // typing in an open dropdown jumps to an option
}

// globalKeys handles Esc (back), F2 (mouse) and ? (help) everywhere. A modal or an open
// dropdown gets Esc itself; Esc in a text field goes back. ? is typed normally in text
// fields.
func (s *Shell) globalKeys(ev *tcell.EventKey) *tcell.EventKey {
	switch {
	case ev.Key() == tcell.KeyF2:
		s.SetMouse(!s.mouse)
		return nil
	case ev.Key() == tcell.KeyEscape:
		if s.modalOpen() || s.dropdownOpen() || s.front() == HomePage {
			return ev
		}
		s.Back()
		return nil
	case ev.Key() == tcell.KeyRune && ev.Rune() == '?':
		if s.modalOpen() || s.typing() {
			return ev
		}
		s.Show(HelpPage, nil)
		return nil
	}
	return ev
}

// dropdownOpen reports whether a dropdown's list is open, which closes on Esc. An open
// dropdown moves the focus to its own *tview.List. Screens don't use plain lists (only
// the sidebar is one), so a focused list that isn't the sidebar is an open dropdown.
func (s *Shell) dropdownOpen() bool {
	switch p := s.App.GetFocus().(type) {
	case *tview.DropDown:
		return p.IsOpen()
	case *tview.List:
		return p != s.Sidebar
	}
	return false
}

const helpText = `
  Keys

  Tab / Shift+Tab   move between fields and buttons
  Arrows            move in lists, menus and tables
  Enter             select, or press the focused button
  Esc               back to the previous screen (closes a dialog first)
  ?                 this help (not while typing in a field)
  F2                mouse on/off. With the mouse off, drag to select text and
                    copy it with your terminal (Ctrl+Shift+C). With it on, hold
                    Shift while dragging.
  q                 quit, from the menu

  Menu shortcuts work while the menu has the focus. Esc gets you back to it.

  Press Esc to close this help.
`

func helpView() tview.Primitive {
	v := tview.NewTextView().SetText(helpText)
	v.SetBorder(true).SetTitle(" Help ").SetTitleAlign(tview.AlignLeft)
	return v
}
