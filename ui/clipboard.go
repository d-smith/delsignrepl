package ui

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// clipboardCommands are tried in order by SystemCopy; the first found on the PATH is
// used. clip.exe is Windows's (and WSL's), pbcopy macOS's, the rest Linux desktops'.
var clipboardCommands = [][]string{
	{"clip.exe"},
	{"pbcopy"},
	{"wl-copy"},
	{"xclip", "-selection", "clipboard"},
	{"xsel", "--clipboard", "--input"},
}

// SystemCopy puts text on the clipboard with the system's clipboard command, returning
// the command's name ("" if none was found). Tests replace it so that they don't touch
// the real clipboard.
var SystemCopy = func(text string) (string, error) {
	for _, c := range clipboardCommands {
		path, err := exec.LookPath(c[0])
		if err != nil {
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, path, c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		return c[0], cmd.Run()
	}
	return "", nil
}

// Copy puts text on the clipboard two ways, since neither always works: through the
// terminal (an OSC 52 escape, which Windows Terminal, iTerm2, kitty and others honour,
// even over ssh), and with the system's clipboard command if there is one. The footer
// says what was copied; what names it there ("txnid", "address 0xAb12...9F3c").
func (s *Shell) Copy(text, what string) {
	if text == "" {
		s.Status("Nothing to copy here.")
		return
	}
	if s.screen != nil {
		s.screen.SetClipboard([]byte(text))
	}
	name, err := SystemCopy(text)
	switch {
	case err != nil:
		s.Status("Copied " + what + " through the terminal; " + name + " failed: " + err.Error())
	case name == "":
		s.Status("Copied " + what + " through the terminal.")
	default:
		s.Status("Copied " + what + ".")
	}
}
