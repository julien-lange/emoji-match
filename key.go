//go:build !(js && wasm)

package main

import (
	"os"
	"os/exec"
	"strings"
)

// waitForKey blocks until the child presses something — any key, no ENTER
// needed. It asks stty to put the terminal in cbreak mode, which delivers each
// keystroke as it happens instead of a line at a time, and turns off echo so
// the key does not appear on screen. Signals are left alone (cbreak, not raw),
// so Ctrl-C still quits.
//
// If cbreak is not available — stdin is a pipe, there is no stty, this is
// Windows — it degrades to reading one byte from whatever arrives, which for a
// terminal means the child presses a key and then ENTER.
func (g *game) waitForKey() {
	restore, raw := cbreak()
	if raw {
		defer restore()
	}

	if _, err := g.in.ReadByte(); err != nil {
		return
	}

	if raw {
		// One key can arrive as several bytes: the arrow keys and friends send
		// an escape sequence. Drop the remainder so it is not read back later
		// as an answer. Only safe in cbreak mode — on a pipe the bytes waiting
		// are the rest of the script, not the tail of a keystroke.
		if n := g.in.Buffered(); n > 0 {
			g.in.Discard(n)
		}
	}
}

// cbreak switches the terminal to single-keypress input and returns a function
// that puts it back exactly as it was.
func cbreak() (restore func(), ok bool) {
	saved, err := stty("-g") // -g prints the current settings in a form stty accepts back
	if err != nil {
		return nil, false
	}
	if _, err := stty("cbreak", "-echo"); err != nil {
		return nil, false
	}
	return func() { stty(saved) }, true
}

func stty(args ...string) (string, error) {
	cmd := exec.Command("stty", args...)
	cmd.Stdin = os.Stdin // stty acts on this terminal, not on its own stdout
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
