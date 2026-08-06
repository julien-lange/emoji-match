//go:build !(js && wasm)

package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// out is where the screen goes. A variable so tests can capture it.
var out io.Writer = os.Stdout

// ANSI escapes. Set to empty strings by disableColour when output is not a
// friendly terminal (or the user passes -nocolor).
var (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[91m"
	green  = "\033[92m"
	yellow = "\033[93m"
	blue   = "\033[94m"
	purple = "\033[95m"
	cyan   = "\033[96m"
)

func disableColour() {
	reset, bold, dim = "", "", ""
	red, green, yellow, blue, purple, cyan = "", "", "", "", "", ""
}

// bigMode turns on double-size text. It uses the VT100 line attributes, which
// are the only way to make an *emoji* bigger in a terminal — a font-based trick
// (block-letter ASCII art) can enlarge letters but not 🐘.
//
//	ESC # 3   top half of a double-height, double-width line
//	ESC # 4   bottom half of the same line
//	ESC # 6   double-width, single-height line
//	ESC # 5   back to a normal line
//
// A double-height line is written twice, once as the top half and once as the
// bottom half; the terminal draws one line of text at twice the size. The cost
// is that a line fits half as many characters, so the layout below is narrower
// in big mode.
var bigMode = true

// tick is the game's entire vocabulary of praise: it appears over a picture and
// over its word the moment they are paired, and nothing else is ever said.
// U+2714 rather than ✅ because a bold green tick has to be coloured by us, and
// ✅ carries its own colour.
func tick() string { return bold + green + "✔" + reset }

func clearScreen() {
	fmt.Fprint(out, "\033[H\033[2J")
}

// emit writes one logical line at double size (or normal size if -big=false).
func emit(s string) {
	if !bigMode {
		fmt.Fprintln(out, s)
		return
	}
	fmt.Fprintf(out, "\033#3%s\n\033#4%s\n", s, s)
}

// emitSmall writes a line at normal size even in big mode. Used for blank
// spacer lines, so vertical space is not wasted doubling nothing.
func emitSmall(s string) {
	if bigMode {
		fmt.Fprint(out, "\033#5")
	}
	fmt.Fprintln(out, s)
}

// emitPrompt writes an instruction at double size, then leaves the cursor on a
// double-width line so the child's own typing shows up twice as wide too.
// (The typing line can only be double-*width*: a double-height line has to be
// printed twice, which is impossible for characters the child has not typed
// yet.)
func emitPrompt(instruction, caret string) {
	emit(instruction)
	if bigMode {
		fmt.Fprint(out, "\033#6")
	}
	fmt.Fprint(out, caret)
}

func indent() string {
	if bigMode {
		return "  "
	}
	return "    "
}

// States for walking an escape sequence in cellWidth.
const (
	textState = iota
	escState
	csiState
	lineAttrState
)

// narrowSymbols are characters that sit inside the wide ranges below but are
// text-presentation by default, so terminals draw them one column wide. The
// tick is the one that matters here.
var narrowSymbols = map[rune]bool{
	0x2713: true, // ✓
	0x2714: true, // ✔
	0x2716: true, // ✖
	0x2717: true, // ✗
	0x2718: true, // ✘
	0x2639: true, // ☹
	0x263A: true, // ☺
}

// cellWidth reports how many terminal columns s occupies. Emoji are two columns
// wide; variation selectors, zero-width joiners and ANSI colour escapes take
// none. Everything in pairs.go is plain ASCII or a double-width emoji, so this
// stays simple. (Double-size lines scale everything equally, so the arithmetic
// is the same in both modes.)
func cellWidth(s string) int {
	w := 0
	state := textState
	for _, r := range s {
		switch state {
		case escState: // just after ESC
			switch r {
			case '[':
				state = csiState
			case '#':
				state = lineAttrState
			default:
				state = textState
			}
			continue
		case csiState: // ESC [ ... runs to its final letter (m, K, J, H...)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				state = textState
			}
			continue
		case lineAttrState: // ESC # n is exactly one character long
			state = textState
			continue
		}
		switch {
		case r == 0x1B:
			state = escState
		case narrowSymbols[r]:
			w++
		case r == 0xFE0F || r == 0xFE0E || r == 0x200D:
			// zero width
		case r >= 0x1F300 && r <= 0x1FAFF,
			r >= 0x1F000 && r <= 0x1F2FF,
			r >= 0x2600 && r <= 0x27BF,
			r == 0x2B50, r == 0x2B55,
			r == 0x231A, r == 0x231B:
			w += 2
		default:
			w++
		}
	}
	return w
}

// centre pads s with spaces so it sits in the middle of a width-column field.
func centre(s string, width int) string {
	pad := width - cellWidth(s)
	if pad <= 0 {
		return s
	}
	left := pad / 2
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", pad-left)
}

// row lays cells out side by side in equal columns of width w.
func row(cells []string, w int) string {
	var b strings.Builder
	b.WriteString(indent())
	for _, c := range cells {
		b.WriteString(centre(c, w))
	}
	return strings.TrimRight(b.String(), " ")
}
