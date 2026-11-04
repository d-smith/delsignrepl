package ui_test

import (
	"delsignrepl/ui"
	"errors"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v2"
)

// fakeClipboard replaces the system clipboard command for the test, recording what's
// copied. name is the command it pretends to be ("" for none found), err its result.
func fakeClipboard(t *testing.T, name string, err error) *[]string {
	t.Helper()
	var got []string
	prev := ui.SystemCopy
	ui.SystemCopy = func(text string) (string, error) {
		got = append(got, text)
		return name, err
	}
	t.Cleanup(func() { ui.SystemCopy = prev })
	return &got
}

// Copy goes through the terminal (OSC 52, which the simulation screen records) and the
// system command, and the footer says how it went.
func TestCopy(t *testing.T) {
	sh, sim, _ := testShell(t)
	got := fakeClipboard(t, "clip.exe", nil)

	sim.Do(func() { sh.Copy("0xabc", "address") })
	if string(sim.Screen.GetClipboardData()) != "0xabc" || len(*got) != 1 || (*got)[0] != "0xabc" {
		t.Errorf("terminal %q, system %q", sim.Screen.GetClipboardData(), *got)
	}
	sim.Shows("Copied address.")

	fakeClipboard(t, "", nil)
	sim.Do(func() { sh.Copy("0xdef", "txnid") })
	sim.Shows("Copied txnid through the terminal.")

	fakeClipboard(t, "xclip", errors.New("no display"))
	sim.Do(func() { sh.Copy("1", "wallet id") })
	sim.Shows("through the terminal; xclip failed: no display")

	sim.Do(func() { sh.Copy("", "nothing") })
	sim.Shows("Nothing to copy here.")
}

// Activity lists entries newest first; Enter shows one in full and c copies its value.
func TestActivityScreen(t *testing.T) {
	sh, sim, _ := testShell(t)
	got := fakeClipboard(t, "", nil)

	sim.Do(sh.ShowActivity)
	sim.Shows("Nothing yet.")
	sim.Key(tcell.KeyEscape)

	sim.Do(func() {
		sh.Log(ui.Entry{What: "Created wallet", Summary: "wallet 7", Copy: "7", CopyIs: "wallet id"})
		sh.Log(ui.Entry{What: "Send: sent", Summary: "0.001 ETH to 0x12", Details: []string{"Txnid      0xfeed"},
			Copy: "0xfeed", CopyIs: "txnid"})
		sh.ShowActivity()
	})
	text := sim.Text()
	sent, wallet := strings.Index(text, "Send: sent"), strings.Index(text, "Created wallet")
	if sent < 0 || wallet < 0 || sent > wallet {
		t.Fatalf("want the send above the wallet (newest first):\n%s", text)
	}

	sim.Type("c")
	if len(*got) != 1 || (*got)[0] != "0xfeed" {
		t.Errorf("c on the newest entry copied %q", *got)
	}
	sim.Shows("Copied txnid")

	sim.Key(tcell.KeyEnter)
	sim.Shows("Txnid      0xfeed")
	if sh.Current() != ui.EntryPage {
		t.Fatalf("Enter opened %q", sh.Current())
	}
	sim.Type("c")
	if len(*got) != 2 || (*got)[1] != "0xfeed" {
		t.Errorf("c on the details copied %q", *got)
	}
	sim.Key(tcell.KeyEscape)
	if sh.Current() != ui.ActivityPage {
		t.Errorf("Esc from the details went to %q", sh.Current())
	}
	sim.Key(tcell.KeyDown)
	sim.Type("c")
	if (*got)[len(*got)-1] != "7" {
		t.Errorf("c on the older entry copied %q", *got)
	}
}
