package main

import (
	"math/rand"
	"slices"
)

// recentMemory is how many recently used pairs are kept off the table, so the
// same emoji does not come round again two turns later.
const recentMemory = 15

// picker draws rounds from the pool without repeating itself too quickly.
type picker struct {
	rng    *rand.Rand
	recent []int // indices into pairs, oldest first
}

// pick returns n pairs for a round. Words within a round always start with
// different letters: a child who can only sound out the first letter still has
// a way in, which is exactly the skill being practised.
func (p *picker) pick(n int) []Pair {
	if chosen, ok := p.attempt(n, true); ok {
		return chosen
	}
	// Pool exhausted by the "recently seen" filter (only possible with a tiny
	// pool or a large n) — try again allowing repeats.
	chosen, _ := p.attempt(n, false)
	return chosen
}

func (p *picker) attempt(n int, avoidRecent bool) ([]Pair, bool) {
	order := p.rng.Perm(len(pairs))
	usedFirst := map[byte]bool{}

	var idxs []int
	for _, i := range order {
		if len(idxs) == n {
			break
		}
		if avoidRecent && p.isRecent(i) {
			continue
		}
		first := pairs[i].Word[0]
		if usedFirst[first] {
			continue
		}
		usedFirst[first] = true
		idxs = append(idxs, i)
	}
	if len(idxs) < n {
		return nil, false
	}

	chosen := make([]Pair, 0, n)
	for _, i := range idxs {
		chosen = append(chosen, pairs[i])
		p.remember(i)
	}
	return chosen, true
}

func (p *picker) isRecent(i int) bool {
	return slices.Contains(p.recent, i)
}

func (p *picker) remember(i int) {
	p.recent = append(p.recent, i)
	if len(p.recent) > recentMemory {
		p.recent = p.recent[len(p.recent)-recentMemory:]
	}
}

// round holds one puzzle: the same pairs laid out in two independent orders,
// emoji across the top and words underneath.
type round struct {
	emojis []Pair // position 1..n
	words  []Pair // position a..n
	solved []bool // indexed by emoji position
	col    int    // column width, sized to this round's longest word
}

func newRound(rng *rand.Rand, chosen []Pair) *round {
	n := len(chosen)

	emojis := make([]Pair, n)
	copy(emojis, chosen)
	rng.Shuffle(n, func(i, j int) { emojis[i], emojis[j] = emojis[j], emojis[i] })

	words := make([]Pair, n)
	copy(words, chosen)
	// Reshuffle while the answer is the giveaway 1a / 2b / 3c, which teaches
	// position rather than reading.
	for range 20 {
		rng.Shuffle(n, func(i, j int) { words[i], words[j] = words[j], words[i] })
		if !sameOrder(emojis, words) {
			break
		}
	}

	// Size the columns to the words actually on screen. Double-size text halves
	// the usable width of the terminal, so a round of "cat / sun / hen" should
	// not be spaced out as if it contained "elephant".
	longest := 0
	for _, p := range chosen {
		longest = max(longest, len(p.Word))
	}
	col := max(longest+2, 8)

	return &round{emojis: emojis, words: words, solved: make([]bool, n), col: col}
}

func sameOrder(a, b []Pair) bool {
	for i := range a {
		if a[i].Word != b[i].Word {
			return false
		}
	}
	return true
}

func (r *round) size() int { return len(r.emojis) }

func (r *round) done() bool {
	for _, s := range r.solved {
		if !s {
			return false
		}
	}
	return true
}

// wordPos finds where a word sits in the word row, given its emoji position.
func (r *round) wordPos(emojiPos int) int {
	for i, w := range r.words {
		if w.Word == r.emojis[emojiPos].Word {
			return i
		}
	}
	return -1
}
