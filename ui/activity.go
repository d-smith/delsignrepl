package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ActivityPage is the Activity screen's page name, and EntryPage an entry's details.
const (
	ActivityPage = "Activity"
	EntryPage    = "Activity entry"
)

// Entry is one thing that happened this session: a wallet, address or key made, a key
// registered, or a send attempt.
type Entry struct {
	Time    time.Time
	What    string   // short: "Send: sent", "Created address"
	Summary string   // one line for the table
	Details []string // the full record, shown on Enter
	Copy    string   // what c copies: a txnid, address, wallet id or path ("" for none)
	CopyIs  string   // names Copy in the footer: "txnid", "address"
}

// Log records e (stamped now if it has no time) in the session's activity. It's kept
// in memory only.
func (s *Shell) Log(e Entry) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	s.activity = append(s.activity, e)
}

// Activity returns the session's entries, oldest first.
func (s *Shell) Activity() []Entry {
	return append([]Entry{}, s.activity...)
}

const activityHint = " Enter details   c copy (txnid, address, ...)   Esc back"

// ShowActivity opens the Activity screen: the session's entries, newest first.
func (s *Shell) ShowActivity() {
	t := tview.NewTable().SetFixed(1, 0).SetSelectable(true, false)
	t.SetBorder(true).SetTitle(" Activity (this session) ").SetTitleAlign(tview.AlignLeft)
	for col, h := range []string{"Time", "What", "Details"} {
		t.SetCell(0, col, tview.NewTableCell(h).SetSelectable(false).
			SetTextColor(tcell.ColorYellow).SetAttributes(tcell.AttrBold))
	}
	entries := s.Activity()
	newest := func(row int) (Entry, bool) {
		i := len(entries) - row
		if row < 1 || i < 0 {
			return Entry{}, false
		}
		return entries[i], true
	}
	if len(entries) == 0 {
		t.SetCell(1, 2, tview.NewTableCell("Nothing yet. Wallets, addresses, keys and sends made "+
			"this session are listed here.").SetSelectable(false))
	}
	for row := 1; row <= len(entries); row++ {
		e, _ := newest(row)
		t.SetCell(row, 0, tview.NewTableCell(e.Time.Format("15:04:05")))
		t.SetCell(row, 1, tview.NewTableCell(e.What))
		t.SetCell(row, 2, tview.NewTableCell(e.Summary).SetExpansion(1))
	}
	if len(entries) > 0 {
		t.Select(1, 0)
	}
	t.SetSelectedFunc(func(row, _ int) {
		if e, ok := newest(row); ok {
			s.showEntry(e)
		}
	})
	t.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'c' {
			row, _ := t.GetSelection()
			if e, ok := newest(row); ok {
				s.Copy(e.Copy, e.CopyIs)
			}
			return nil
		}
		return ev
	})
	hint := tview.NewTextView().SetText(activityHint).SetTextColor(tcell.ColorGray)
	s.Show(ActivityPage, tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(t, 0, 1, true).
		AddItem(hint, 1, 0, false))
}

// showEntry shows e in full on its own screen, one value per line, so that long values
// (txnids, addresses) aren't wrapped and can be drag-selected. c copies, Esc goes back.
func (s *Shell) showEntry(e Entry) {
	text := "\n  " + e.Time.Format("15:04:05") + "  " + e.What + "\n\n"
	for _, d := range e.Details {
		text += "  " + d + "\n"
	}
	v := tview.NewTextView().SetText(tview.Escape(text))
	v.SetBorder(true).SetTitle(" " + e.What + " ").SetTitleAlign(tview.AlignLeft)
	v.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		if ev.Key() == tcell.KeyRune && ev.Rune() == 'c' {
			s.Copy(e.Copy, e.CopyIs)
			return nil
		}
		return ev
	})
	hint := tview.NewTextView().SetText(" c copy   Esc back").SetTextColor(tcell.ColorGray)
	s.Show(EntryPage, tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(v, 0, 1, true).
		AddItem(hint, 1, 0, false))
}
