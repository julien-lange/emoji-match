package main

import (
	"bufio"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type stats struct {
	rounds   int
	matches  int
	firstTry int
}

type game struct {
	in    *bufio.Reader
	rng   *rand.Rand
	pick  *picker
	perRd int
	maxRd int
	st    stats
}

func main() {
	n := flag.Int("n", 3, "how many emoji/word pairs per round (2-5)")
	rounds := flag.Int("rounds", 0, "stop after this many rounds (0 = play until you quit)")
	noColour := flag.Bool("nocolor", false, "turn off colours")
	big := flag.Bool("big", true, "double-size text and emoji (needs a VT100-compatible terminal)")
	seed := flag.Int64("seed", 0, "random seed (0 = pick one from the clock)")
	flag.Parse()

	if *noColour || os.Getenv("NO_COLOR") != "" {
		disableColour()
	}
	bigMode = *big
	*n = max(2, min(*n, 5))

	s := *seed
	if s == 0 {
		s = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(s))

	g := &game{
		in:    bufio.NewReader(os.Stdin),
		rng:   rng,
		pick:  &picker{rng: rng},
		perRd: *n,
		maxRd: *rounds,
	}
	g.run()
}

func (g *game) run() {
	if !g.intro() {
		return
	}
	for g.maxRd == 0 || g.st.rounds < g.maxRd {
		if !g.playRound() {
			break
		}
	}
	g.summary()
}

// ---------------------------------------------------------------- screens ---

// intro is the only screen aimed at the grown-up, so it stays at normal size —
// it has more words on it than a double-size line can hold.
func (g *game) intro() bool {
	clearScreen()
	ind := "    "

	emitSmall("")
	emitSmall(fmt.Sprintf("%s🎈 %sE M O J I   M A T C H%s", ind, bold+purple, reset))
	emitSmall("")
	emitSmall(fmt.Sprintf("%sMatch every picture to the word that says its name.", ind))
	emitSmall("")
	emitSmall(fmt.Sprintf("%s%sHow to play:%s", ind, bold, reset))
	emitSmall(fmt.Sprintf("%s  1. Look at a picture. It has a %snumber%s above it.", ind, cyan, reset))
	emitSmall(fmt.Sprintf("%s  2. Find its word. Each word has a %sletter%s above it.", ind, yellow, reset))
	emitSmall(fmt.Sprintf("%s  3. Type them together — %s1a%s or %sa1%s, either way round.",
		ind, bold+green, reset, bold+green, reset))
	emitSmall(fmt.Sprintf("%s  4. Press the big %sENTER%s key.", ind, bold, reset))
	emitSmall("")

	var nums, lets []string
	for i := range g.perRd {
		nums = append(nums, fmt.Sprintf("%d", i+1))
		lets = append(lets, string(rune('a'+i)))
	}
	emit(fmt.Sprintf("%s%s%s   %s%s%s   %sENTER%s",
		ind, cyan, strings.Join(nums, " "), yellow, strings.Join(lets, " "), reset, bold, reset))

	emitSmall("")
	emitSmall(fmt.Sprintf("%s%sType q and ENTER to stop.%s", ind, dim, reset))
	if bigMode {
		emitSmall(fmt.Sprintf("%s%sSeeing every line twice? Your terminal can't do big"+
			" text — rerun with -big=false.%s", ind, dim, reset))
	}
	emitSmall("")
	fmt.Fprintf(out, "%s%sPress ENTER to start!%s ", ind, bold+green, reset)

	_, ok := g.readLine()
	return ok
}

func (g *game) playRound() bool {
	r := newRound(g.rng, g.pick.pick(g.perRd))
	g.st.rounds++

	attempts := make([]int, r.size()) // wrong tries against each emoji this round

	for !r.done() {
		g.draw(r)
		emitPrompt(
			fmt.Sprintf("%sType %s1a%s or %sa1%s, then ENTER:", indent(), bold+green, reset, bold+green, reset),
			indent()+"> ")

		line, ok := g.readLine()
		if !ok || isQuit(line) {
			return false
		}

		ePos, wPos, parsed := parseGuess(line)
		// Anything that is not a usable answer — a stray keypress, a number or
		// letter that is not on screen, a picture already ticked — is simply
		// not an answer, so it draws no reaction at all. Only a real wrong
		// match gets the sad face.
		if !parsed || ePos >= r.size() || wPos >= r.size() || r.solved[ePos] {
			continue
		}

		if r.wordPos(ePos) == wPos {
			r.solved[ePos] = true
			g.st.matches++
			if attempts[ePos] == 0 {
				g.st.firstTry++
			}
			continue
		}

		attempts[ePos]++
		g.sadFace()
	}

	g.draw(r)

	if g.maxRd > 0 && g.st.rounds >= g.maxRd {
		return true
	}
	emitPrompt(fmt.Sprintf("%sENTER to keep playing, q to stop:", indent()), indent()+"> ")
	line, ok := g.readLine()
	return ok && !isQuit(line)
}

func (g *game) summary() {
	ind := indent()
	emitSmall("")
	emit(fmt.Sprintf("%s🎉 %sThanks for playing!%s", ind, bold+purple, reset))
	emitSmall("")
	emit(fmt.Sprintf("%sRounds:      %s%d%s", ind, bold, g.st.rounds, reset))
	emit(fmt.Sprintf("%sMatched:     %s%d%s", ind, bold, g.st.matches, reset))
	emit(fmt.Sprintf("%sFirst time:  %s%d%s", ind, bold+green, g.st.firstTry, reset))
	if g.st.firstTry > 0 {
		maxStars := 20
		if bigMode {
			maxStars = 10 // a double-size star is four columns of screen
		}
		emit(ind + strings.Repeat("⭐", min(g.st.firstTry, maxStars)))
	}
	emitSmall("")
	emit(fmt.Sprintf("%s%sCome back soon! 👋%s", ind, cyan, reset))
	emitSmall("")
}

// ---------------------------------------------------------------- drawing ---

func (g *game) draw(r *round) {
	clearScreen()
	emitSmall("")

	emit(fmt.Sprintf("%s🎈 %sEMOJI MATCH%s  %sRound %d%s  %s⭐ %d%s",
		indent(), bold+purple, reset, dim, g.st.rounds, reset, yellow, g.st.firstTry, reset))
	emitSmall("")

	emit(row(g.labels(r, true), r.col))
	// The boxes are a nice frame at normal size; in big mode the emoji is
	// already unmissable and the four extra screen rows are better spent on
	// the words.
	if !bigMode {
		emit(row(g.boxCells(r, "┌─────┐"), r.col))
	}
	emit(row(g.emojiCells(r), r.col))
	if !bigMode {
		emit(row(g.boxCells(r, "└─────┘"), r.col))
	}
	emitSmall("")

	emit(row(g.labels(r, false), r.col))
	emit(row(g.wordCells(r), r.col))
	emitSmall("")
}

// sadFace is the whole of the negative feedback: no words, no explanation of
// what went wrong, just a sad face that stays up until the child presses a key.
func (g *game) sadFace() {
	clearScreen()
	emitSmall("")
	emitSmall("")
	emitSmall("")
	emit(indent() + "🙁")
	g.waitForKey()
}

// labels builds the row of numbers (over the pictures) or letters (over the
// words). A solved item shows a tick instead — it no longer needs choosing,
// and it saves a whole row of screen for the tick marks.
func (g *game) labels(r *round, forEmojis bool) []string {
	cells := make([]string, r.size())
	for i := range cells {
		var solved bool
		var text, colour string
		if forEmojis {
			solved, text, colour = r.solved[i], fmt.Sprintf("%d", i+1), cyan
		} else {
			solved, text, colour = g.wordSolved(r, i), string(rune('a'+i)), yellow
		}
		if solved {
			cells[i] = tick()
		} else {
			cells[i] = colour + bold + text + reset
		}
	}
	return cells
}

func (g *game) emojiCells(r *round) []string {
	cells := make([]string, r.size())
	for i, e := range r.emojis {
		if bigMode {
			cells[i] = e.Emoji // no frame in big mode, see draw
			continue
		}
		colour := blue
		if r.solved[i] {
			colour = green
		}
		cells[i] = colour + "│" + reset + centre(e.Emoji, 5) + colour + "│" + reset
	}
	return cells
}

func (g *game) boxCells(r *round, art string) []string {
	cells := make([]string, r.size())
	for i := range cells {
		colour := blue
		if r.solved[i] {
			colour = green
		}
		cells[i] = colour + art + reset
	}
	return cells
}

func (g *game) wordCells(r *round) []string {
	cells := make([]string, r.size())
	for i, w := range r.words {
		if g.wordSolved(r, i) {
			cells[i] = dim + w.Word + reset
		} else {
			cells[i] = bold + w.Word + reset
		}
	}
	return cells
}

func (g *game) wordSolved(r *round, wordPos int) bool {
	for i := range r.emojis {
		if r.solved[i] && r.wordPos(i) == wordPos {
			return true
		}
	}
	return false
}

// ------------------------------------------------------------------ input ---

// readLine reads one typed line. A bufio.Reader rather than a Scanner, because
// waitForKey needs to take a single byte from the same stream and to know
// whether anything is still buffered.
func (g *game) readLine() (string, bool) {
	line, err := g.in.ReadString('\n')
	if err != nil && line == "" {
		return "", false // Ctrl-D, or stdin ran out
	}
	return strings.TrimRight(line, "\r\n"), true
}

func isQuit(line string) bool {
	s := strings.ToLower(strings.TrimSpace(line))
	return s == "q" || s == "quit" || s == "exit" || s == "stop"
}

// parseGuess pulls a picture number and a word letter out of whatever the child
// typed, in either order: "1a" and "a1" both mean picture 1 with word a. It is
// deliberately forgiving — "1 A" and "1-a" work too, and anything else in the
// line is ignored. Returned positions are zero-based.
func parseGuess(line string) (emojiPos, wordPos int, ok bool) {
	emojiPos, wordPos = -1, -1
	for _, r := range strings.ToLower(line) {
		switch {
		case r >= '1' && r <= '9' && emojiPos < 0:
			emojiPos = int(r - '1')
		case r >= 'a' && r <= 'z' && wordPos < 0:
			wordPos = int(r - 'a')
		}
	}
	return emojiPos, wordPos, emojiPos >= 0 && wordPos >= 0
}
