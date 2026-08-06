package main

import (
	"bufio"
	"math/rand"
	"os"
	"strings"
	"testing"
)

func TestCellWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"dog", 3},
		{"🐶", 2},
		{"\033[92mdog\033[0m", 3},
		{"\033[1m\033[92m🐶\033[0m", 2},
		{"\033[35mdog\033[0m", 3},      // a colour code containing a digit the ESC # form uses
		{"\033#3dog", 3},               // VT100 line attribute, one character long
		{"\033[1m\033[92m✔\033[0m", 1}, // text-presentation tick, one column
		{"", 0},
	}
	for _, c := range cases {
		if got := cellWidth(c.in); got != c.want {
			t.Errorf("cellWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestCentreIgnoresColour(t *testing.T) {
	plain := centre("dog", 11)
	fancy := centre("\033[1mdog\033[0m", 11)
	if cellWidth(plain) != 11 || cellWidth(fancy) != 11 {
		t.Fatalf("centred cells are %d and %d columns, want 11", cellWidth(plain), cellWidth(fancy))
	}
}

func TestParseGuess(t *testing.T) {
	cases := []struct {
		in    string
		emoji int
		word  int
		ok    bool
	}{
		{"1a", 0, 0, true},
		{"a1", 0, 0, true},
		{"3 C", 2, 2, true},
		{"2-b", 1, 1, true},
		{"  2b  ", 1, 1, true},
		{"1", -1, -1, false},
		{"b", -1, -1, false},
		{"", -1, -1, false},
		{"!!", -1, -1, false},
	}
	for _, c := range cases {
		e, w, ok := parseGuess(c.in)
		if ok != c.ok || (ok && (e != c.emoji || w != c.word)) {
			t.Errorf("parseGuess(%q) = (%d,%d,%v), want (%d,%d,%v)", c.in, e, w, ok, c.emoji, c.word, c.ok)
		}
	}
}

// Every emoji in the pool must be double-width, or the columns stop lining up.
// This catches emoji that need a U+FE0F variation selector to be colourful.
func TestPairsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, p := range pairs {
		if w := cellWidth(p.Emoji); w != 2 {
			t.Errorf("emoji %q for %q is %d columns wide, want 2", p.Emoji, p.Word, w)
		}
		if seen[p.Word] {
			t.Errorf("duplicate word %q", p.Word)
		}
		seen[p.Word] = true
		for _, r := range p.Word {
			if r < 'a' || r > 'z' {
				t.Errorf("word %q should be a single lower-case word", p.Word)
				break
			}
		}
	}
}

func TestPickGivesDistinctFirstLetters(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	p := &picker{rng: rng}
	for range 200 {
		chosen := p.pick(3)
		if len(chosen) != 3 {
			t.Fatalf("picked %d pairs, want 3", len(chosen))
		}
		first := map[byte]bool{}
		for _, c := range chosen {
			if first[c.Word[0]] {
				t.Fatalf("two words start with %q in %v", string(c.Word[0]), chosen)
			}
			first[c.Word[0]] = true
		}
	}
}

func TestRoundMappingSurvivesShuffling(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	p := &picker{rng: rng}
	for range 200 {
		r := newRound(rng, p.pick(3))
		for i, e := range r.emojis {
			pos := r.wordPos(i)
			if pos < 0 || r.words[pos].Word != e.Word {
				t.Fatalf("emoji %q at slot %d does not map back to its word", e.Emoji, i)
			}
		}
	}
}

func TestRoundIsNotTheIdentityLayout(t *testing.T) {
	rng := rand.New(rand.NewSource(3))
	p := &picker{rng: rng}
	identity := 0
	for range 200 {
		r := newRound(rng, p.pick(3))
		if sameOrder(r.emojis, r.words) {
			identity++
		}
	}
	if identity > 0 {
		t.Errorf("%d rounds had the giveaway 1a/2b/3c layout", identity)
	}
}

// Big mode leans on the VT100 line attributes: a double-height line has to be
// written twice, as a top half and a bottom half.
func TestEmitDoubleSize(t *testing.T) {
	var buf strings.Builder
	out = &buf
	defer func() { out = os.Stdout }()

	bigMode = true
	emit("hi")
	if got, want := buf.String(), "\033#3hi\n\033#4hi\n"; got != want {
		t.Errorf("big emit = %q, want %q", got, want)
	}

	buf.Reset()
	emitSmall("hi")
	if got, want := buf.String(), "\033#5hi\n"; got != want {
		t.Errorf("big emitSmall = %q, want %q", got, want)
	}

	buf.Reset()
	emitPrompt("type:", "> ")
	if got, want := buf.String(), "\033#3type:\n\033#4type:\n\033#6> "; got != want {
		t.Errorf("big emitPrompt = %q, want %q", got, want)
	}

	buf.Reset()
	bigMode = false
	defer func() { bigMode = true }()
	emit("hi")
	if got, want := buf.String(), "hi\n"; got != want {
		t.Errorf("normal emit = %q, want %q", got, want)
	}
}

// Double-size text halves the usable width of the terminal, so columns are
// sized to the words actually in the round rather than the longest possible.
func TestRoundColumnWidthFollowsWords(t *testing.T) {
	short := newRound(rand.New(rand.NewSource(1)),
		[]Pair{{"🐱", "cat"}, {"🌞", "sun"}, {"🐔", "hen"}})
	long := newRound(rand.New(rand.NewSource(1)),
		[]Pair{{"🐘", "elephant"}, {"🌞", "sun"}, {"🐔", "hen"}})

	if short.col != 8 {
		t.Errorf("short round column = %d, want 8", short.col)
	}
	if long.col != 10 {
		t.Errorf("long round column = %d, want 10", long.col)
	}
	// Every cell is padded to the same width, whatever it holds, so the picture
	// row and the word row line up on screen. (row trims the trailing spaces,
	// hence cells that exactly fill their column.)
	full := strings.Repeat("x", long.col)
	line := row([]string{full, full, full}, long.col)
	if got, want := cellWidth(line), len(indent())+3*long.col; got != want {
		t.Errorf("row width = %d columns, want %d", got, want)
	}
}

// The tick is the only praise in the game, and it stands where a number or a
// letter stood, so it has to be one column wide like they are.
func TestTickIsOneColumn(t *testing.T) {
	if got := cellWidth(tick()); got != 1 {
		t.Errorf("cellWidth(tick()) = %d, want 1", got)
	}
}

// Negative feedback is a face and nothing else — no words, no explanation of
// which pair was wrong.
func TestSadFaceSaysNothing(t *testing.T) {
	var buf strings.Builder
	out = &buf
	defer func() { out = os.Stdout }()

	g := &game{in: bufio.NewReader(strings.NewReader("x"))}
	g.sadFace()

	got := buf.String()
	if !strings.Contains(got, "🙁") {
		t.Errorf("sad face screen %q does not show a face", got)
	}
	visible := stripEscapes(got)
	if strings.ContainsFunc(visible, func(c rune) bool {
		return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
	}) {
		t.Errorf("sad face screen shows words: %q", visible)
	}
}

// stripEscapes removes ANSI colour sequences and VT100 line attributes, leaving
// what the child actually sees.
func stripEscapes(s string) string {
	var b strings.Builder
	state := textState
	for _, r := range s {
		switch state {
		case escState:
			switch r {
			case '[':
				state = csiState
			case '#':
				state = lineAttrState
			default:
				state = textState
			}
		case csiState:
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				state = textState
			}
		case lineAttrState:
			state = textState
		default:
			if r == 0x1B {
				state = escState
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// A wrong answer must not be confused with a keystroke that is not an answer at
// all: only a real mismatch counts against the child.
func TestOnlyRealMismatchesAreWrong(t *testing.T) {
	r := newRound(rand.New(rand.NewSource(1)),
		[]Pair{{"🐱", "cat"}, {"🌞", "sun"}, {"🐔", "hen"}})

	for _, in := range []string{"", "zz", "9z", "1z", "4a"} {
		e, w, parsed := parseGuess(in)
		if parsed && e < r.size() && w < r.size() {
			t.Errorf("%q should not be a usable answer", in)
		}
	}

	// The right pairing for picture 1, found the long way round.
	want := r.wordPos(0)
	e, w, parsed := parseGuess("1" + string(rune('a'+want)))
	if !parsed || e != 0 || w != want {
		t.Fatalf("parseGuess of the correct answer failed: %d %d %v", e, w, parsed)
	}
}

func TestIsQuit(t *testing.T) {
	for _, s := range []string{"q", "Q", " quit ", "exit", "STOP"} {
		if !isQuit(s) {
			t.Errorf("isQuit(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "1a", "queen"} {
		if isQuit(s) {
			t.Errorf("isQuit(%q) = true, want false", s)
		}
	}
}
