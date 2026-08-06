//go:build js && wasm

// This file is the whole of the WebAssembly build. It exports one function to
// the page and shares pairs.go and round.go with the terminal game, so there is
// only ever one set of rules about what makes a round: the same curated pool,
// the same "no pair twice in fifteen draws", the same distinct first letters,
// the same guarantee that the answer is never the giveaway 1a/2b/3c.
package main

import (
	"encoding/json"
	"math/rand"
	"syscall/js"
	"time"
)

// One picker for the life of the page. The "recently seen" memory is what stops
// 🐶 coming back two rounds later, and it only means anything if it survives
// between calls — a picker made fresh each time would remember nothing.
var session = &picker{rng: rand.New(rand.NewSource(time.Now().UnixNano()))}

// wireRound is one puzzle as the page reads it. Emojis and words are in two
// independent orders, exactly as in round.go, and Answer carries the mapping
// between them: Answer[i] is where emoji i's word sits in Words.
//
// The mapping travels with the round rather than being recomputed in the page,
// so the browser never needs its own copy of the pool to check an answer.
type wireRound struct {
	N      int      `json:"n"`
	Emojis []string `json:"emojis"`
	Words  []string `json:"words"`
	Answer []int    `json:"answer"`
}

func main() {
	// Set before main blocks, so the export exists by the time go.run() hands
	// control back to the page.
	js.Global().Set("emojiRound", js.FuncOf(generate))
	select {} // syscall/js callbacks only run while main is alive
}

// generate implements window.emojiRound(n, [seed]) and returns one round as a
// JSON string.
//
// Without a seed it draws from the session picker, which is what the page wants:
// endless rounds that do not repeat themselves. With one it builds a throwaway
// picker instead, so the same seed always gives the same round — that is for the
// tests, not for the child.
func generate(_ js.Value, args []js.Value) any {
	// A number is taken and clamped, so a page asking for something silly gets a
	// playable round rather than an error. Anything else — no argument at all,
	// most likely — means the default. Hence the type check rather than the
	// truthiness test used for the seed below: zero is a bad answer, not a
	// missing one.
	n := 3
	if len(args) > 0 && args[0].Type() == js.TypeNumber {
		n = args[0].Int()
	}
	n = max(2, min(n, 5))

	p, rng := session, session.rng
	if len(args) > 1 && args[1].Truthy() {
		rng = rand.New(rand.NewSource(int64(args[1].Float())))
		p = &picker{rng: rng}
	}

	r := newRound(rng, p.pick(n))
	if r.size() != n {
		return `{"error":"could not pick that many pairs"}`
	}

	w := wireRound{
		N:      n,
		Emojis: make([]string, n),
		Words:  make([]string, n),
		Answer: make([]int, n),
	}
	for i, e := range r.emojis {
		w.Emojis[i] = e.Emoji
		w.Answer[i] = r.wordPos(i)
	}
	for i, x := range r.words {
		w.Words[i] = x.Word
	}

	b, err := json.Marshal(w)
	if err != nil {
		return `{"error":"encoding failed"}`
	}
	return string(b)
}
