package ui

import (
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Unwrapper is a primitive that wraps another. The shell looks through it to decide
// whether the focus is a text field.
type Unwrapper interface {
	Unwrap() tview.Primitive
}

// TrimmedField is an input field that trims spaces and newlines from pasted text, so a
// pasted line with a trailing newline isn't refused by the field's acceptance function.
type TrimmedField struct {
	*tview.InputField
}

// TrimPaste wraps f so that pastes are trimmed.
func TrimPaste(f *tview.InputField) *TrimmedField {
	return &TrimmedField{f}
}

// PasteHandler trims the pasted text, then pastes it as the input field would.
func (f *TrimmedField) PasteHandler() func(pastedText string, setFocus func(p tview.Primitive)) {
	inner := f.InputField.PasteHandler()
	return func(text string, setFocus func(p tview.Primitive)) {
		inner(strings.TrimSpace(text), setFocus)
	}
}

// Unwrap returns the input field.
func (f *TrimmedField) Unwrap() tview.Primitive { return f.InputField }

func unwrap(p tview.Primitive) tview.Primitive {
	for {
		u, ok := p.(Unwrapper)
		if !ok {
			return p
		}
		p = u.Unwrap()
	}
}

// LiveText is a text view that shows f's result, recomputed each time it's drawn, so it
// never goes stale whatever changed the session. The text is set only when it changes,
// so redraws don't reset the scroll position.
type LiveText struct {
	*tview.TextView
	f    func() string
	last string
	set  bool
}

// NewLiveText returns a LiveText showing f's result.
func NewLiveText(f func() string) *LiveText {
	return &LiveText{TextView: tview.NewTextView(), f: f}
}

// Draw updates the text, then draws the view.
func (l *LiveText) Draw(screen tcell.Screen) {
	if text := l.f(); !l.set || text != l.last {
		l.last, l.set = text, true
		l.TextView.SetText(tview.Escape(text))
	}
	l.TextView.Draw(screen)
}
