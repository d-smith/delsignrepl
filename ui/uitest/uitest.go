// Package uitest runs a tview application on tcell's simulation screen, for tests: inject
// keys, wait for the app to settle, and read what's on screen.
package uitest

import (
	"strings"
	"sync/atomic"
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

	sent int64        // key events injected
	seen atomic.Int64 // key events the app's event loop has taken
}

// Start runs run (which must call app.Run, directly or not) on a w x h simulation screen
// and stops the app when the test ends.
func Start(t *testing.T, app *tview.Application, w, h int, run func() error) *Sim {
	t.Helper()
	scr := tcell.NewSimulationScreen("UTF-8")
	app.SetScreen(scr)
	scr.SetSize(w, h)
	s := &Sim{t: t, App: app, Screen: scr}

	// Count key events as the app takes them, so Key and Type can wait for each one to
	// be handled. tview reads keys and queued updates from separate channels, so an
	// update round trip alone doesn't prove a typed key has been handled.
	prev := app.GetInputCapture()
	app.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		s.seen.Add(1)
		if prev != nil {
			return prev(ev)
		}
		return ev
	})

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

// Key injects a special key (Esc, Enter, Tab...), waits for it to be handled, and
// settles.
func (s *Sim) Key(k tcell.Key) {
	s.t.Helper()
	s.inject(k, 0)
	s.waitHandled()
}

// Type injects each rune of text as a key press, waits for them all to be handled, and
// settles.
func (s *Sim) Type(text string) {
	s.t.Helper()
	for _, r := range text {
		s.inject(tcell.KeyRune, r)
	}
	s.waitHandled()
}

func (s *Sim) inject(k tcell.Key, r rune) {
	s.sent++
	s.Screen.InjectKey(k, r, tcell.ModNone)
}

// waitHandled waits until the app has taken every injected key, then settles: the
// update queued by Settle runs after the last key's handler has returned.
func (s *Sim) waitHandled() {
	s.t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for s.seen.Load() < s.sent {
		if time.Now().After(deadline) {
			s.t.Fatalf("app took %d of %d keys", s.seen.Load(), s.sent)
		}
		time.Sleep(time.Millisecond)
	}
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

// Paste sends text as one bracketed paste (paste start, the keys, paste end), as a
// terminal does with paste enabled, and waits until cond holds.
func (s *Sim) Paste(text string, cond func() bool) {
	s.t.Helper()
	s.Screen.PostEvent(tcell.NewEventPaste(true))
	for _, r := range text {
		k := tcell.KeyRune
		if r == '\n' {
			k, r = tcell.KeyEnter, 0
		}
		s.Screen.InjectKey(k, r, tcell.ModNone)
	}
	s.Screen.PostEvent(tcell.NewEventPaste(false))
	deadline := time.Now().Add(5 * time.Second)
	for {
		s.Settle()
		ok := false
		s.Do(func() { ok = cond() })
		if ok {
			return
		}
		if time.Now().After(deadline) {
			s.t.Fatalf("paste of %q didn't take effect", text)
		}
		time.Sleep(5 * time.Millisecond)
	}
}
