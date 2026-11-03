// Package uitest runs a tview application on tcell's simulation screen, for tests: inject
// keys, wait for the app to settle, and read what's on screen.
package uitest

import (
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Sim is a running app on a simulation screen.
type Sim struct {
	t      *testing.T
	App    *tview.Application
	Screen tcell.SimulationScreen
}

// Start runs run (which must call app.Run, directly or not) on a w x h simulation screen
// and stops the app when the test ends.
func Start(t *testing.T, app *tview.Application, w, h int, run func() error) *Sim {
	t.Helper()
	scr := tcell.NewSimulationScreen("UTF-8")
	app.SetScreen(scr)
	scr.SetSize(w, h)
	done := make(chan error, 1)
	go func() { done <- run() }()
	t.Cleanup(func() {
		app.Stop()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("app didn't stop")
		}
	})
	s := &Sim{t: t, App: app, Screen: scr}
	s.Settle()
	return s
}

// Settle waits until every queued event and update has run and the screen is redrawn.
func (s *Sim) Settle() {
	s.t.Helper()
	ch := make(chan struct{})
	s.App.QueueUpdateDraw(func() {})
	s.App.QueueUpdate(func() { close(ch) })
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		s.t.Fatal("app didn't settle")
	}
}

// Do runs f on the app's goroutine, then settles.
func (s *Sim) Do(f func()) {
	s.t.Helper()
	s.App.QueueUpdate(f)
	s.Settle()
}

// Key injects a special key (Esc, Enter, Tab...) and settles.
func (s *Sim) Key(k tcell.Key) {
	s.t.Helper()
	s.Screen.InjectKey(k, 0, tcell.ModNone)
	time.Sleep(10 * time.Millisecond) // the key travels through tcell's event queue
	s.Settle()
}

// Type injects each rune of text as a key press and settles.
func (s *Sim) Type(text string) {
	s.t.Helper()
	for _, r := range text {
		s.Screen.InjectKey(tcell.KeyRune, r, tcell.ModNone)
	}
	time.Sleep(10 * time.Millisecond)
	s.Settle()
}

// Text returns the screen as lines of text.
func (s *Sim) Text() string {
	cells, w, h := s.Screen.GetContents()
	var b strings.Builder
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := cells[y*w+x]
			if len(c.Runes) == 0 {
				b.WriteRune(' ')
			} else {
				b.WriteRune(c.Runes[0])
			}
		}
		b.WriteRune('\n')
	}
	return b.String()
}

// Shows fails the test unless every want appears on screen.
func (s *Sim) Shows(want ...string) {
	s.t.Helper()
	text := s.Text()
	for _, w := range want {
		if !strings.Contains(text, w) {
			s.t.Errorf("screen doesn't show %q:\n%s", w, text)
		}
	}
}
