package ui

import (
	"strings"

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
