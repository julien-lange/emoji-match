//go:build !(js && wasm)

package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// The web app gets its rounds from the WebAssembly build of this same program,
// which is the only generator worth having: one pool, one set of rules, no copy
// to drift out of step.
//
// But a 2 MB wasm binary is a big thing to depend on absolutely, and a child
// tapping a home-screen icon on a train should not be met with a blank page
// because it failed to arrive. So the pool travels beside it as plain JSON too,
// and the page carries a small fallback picker of its own for that one case.
// This is what writes the JSON, straight from the pairs slice.
func writePairs(dest string) {
	type wirePair struct {
		Emoji string `json:"emoji"`
		Word  string `json:"word"`
	}

	out := struct {
		Pairs []wirePair `json:"pairs"`
	}{Pairs: make([]wirePair, len(pairs))}
	for i, p := range pairs {
		out.Pairs[i] = wirePair{Emoji: p.Emoji, Word: p.Word}
	}

	b, err := json.Marshal(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "emoji-match:", err)
		os.Exit(1)
	}
	b = append(b, '\n')
	if err := os.WriteFile(dest, b, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "emoji-match:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "emoji-match: wrote %s, %d pairs\n", dest, len(pairs))
}
