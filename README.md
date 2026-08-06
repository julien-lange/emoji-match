# Emoji Match

A tiny game for 4–5 year olds. Each round shows three pictures and three words,
shuffled independently, and the child pairs them up.

It comes two ways, from one set of rules:

- **In a terminal**, typing a number and a letter — `1a`, or `a1`, whichever
  order they reach for — so they practise reading *and* find their way around a
  keyboard. Everything is drawn at double size, emoji included.
- **On a phone**, drawing a line from each picture to its word with a finger.

`pairs.go` and `round.go` are shared between the two; `main.go` is the terminal
half and `wasm.go` the browser half, each fenced off from the other by a build
tag.

## Run it

```sh
go run .          # or: go build -o emoji-match . && ./emoji-match
make serve        # the phone version, at http://localhost:8732/
```

## Options

| Flag | Meaning |
| --- | --- |
| `-n 3` | pairs per round, 2–5 (start at `-n 2` for a first session) |
| `-big` | double-size text and emoji, on by default; `-big=false` for normal |
| `-rounds 0` | stop after N rounds; 0 plays until you type `q` |
| `-nocolor` | plain text (also honours `NO_COLOR`) |
| `-seed 0` | fix the randomness, handy for testing |
| `-pairs FILE` | write the word pool to FILE as JSON and exit (see below) |

## The phone version

`docs/` is a self-contained static site: no build step, no framework, no
network calls beyond its own files.

    make            # rebuild docs/match.wasm and docs/pairs.json
    make serve      # then open http://localhost:8732/

Pictures down the left, words down the right, and a line drawn between them
with a finger. A card can also be tapped and its partner tapped after it —
small children reach for a tap before they reach for a drag, and both mean the
same answer.

The feedback is the terminal game's, moved to a screen, and there are still only
three answers it can give:

- **Right** — a tick on both cards, the line locks in a colour of its own, and a
  few sparks come out of the word. Lines are coloured by the word they land on,
  so two of them crossing are still two lines.
- **Wrong** — the screen dims to a single 🙁 for a moment, with the line they
  drew showing faintly red behind it. It does not say which pair was wrong.
- **Neither** — a line let go over empty space, or over a card already ticked,
  draws no reaction at all. That was never an answer.

The only words on screen are the ones being read. Everything else is a glyph:
🎈 and a star count in the header, ↺ and → for the buttons. The small
monospace line above the buttons is for you, not for them — which round this is,
how many pairs, and where the round came from.

Leave a round untouched for a few seconds and rings pulse out of the pictures
still waiting, which is the whole of the tutorial.

### Where the rounds come from

`match.wasm` — the same Go generator the terminal game uses, compiled with
`GOOS=js GOARCH=wasm`. It exports one function:

```js
window.emojiRound(n, seed)   // -> JSON: {n, emojis, words, answer}
```

`answer[i]` is where emoji `i`'s word sits in `words`, so the page can mark an
answer without knowing anything about the pool. The seed is for the tests; the
page leaves it out and gets the session picker, which is what keeps a pair from
coming round again two turns later. Nothing is generated ahead of time, so the
game never runs out.

Beside it, `pairs.json` carries the pool as plain text — `make pairs`, which is
`go run . -pairs docs/pairs.json`. A 2.8 MB binary is a big thing to depend on
absolutely, and a child tapping a home-screen icon on a train should not be met
with a blank page, so `app.js` keeps a small fallback picker for that one case.
It also draws the very first round of a session while the wasm is still
downloading, which is why the game starts instantly. `test/player.js` holds it
to the same rules the Go generator keeps.

The footnote line says which one is in use: `wasm` or `json`.

### Publishing to GitHub Pages

Push, then in **Settings → Pages** set the source to **main / docs**. Nothing
else is needed: Pages serves `.wasm` with the right `application/wasm` MIME
type, and `.nojekyll` keeps Jekyll's hands off the directory.

The page installs to a phone home screen ("Add to Home Screen") and works with
no signal afterwards, because `sw.js` caches everything on first visit. **That
cache is why you must bump `VERSION` in `docs/sw.js` whenever you republish a
new wasm build or a changed pool** — otherwise phones will keep serving the old
one.

## Big text

Terminals have no "font size" a program can set, and no amount of ASCII art can
enlarge 🐘. What does work is the VT100 line attributes, which scale everything
on a line — letters and emoji alike:

- `ESC # 3` / `ESC # 4` — top and bottom half of a double-height, double-width
  line. The same text is printed twice; the terminal draws it once, at twice
  the size.
- `ESC # 6` — double-width, single-height. Used for the line the child types
  on, since the other trick needs text printed twice and their keystrokes have
  not happened yet.

Supported by Terminal.app, iTerm2, xterm, kitty, WezTerm and Ghostty. If your
terminal does not support it you will see every line printed twice — rerun with
`-big=false`. The intro screen says so too.

Big text halves how much fits on a line, so in big mode the frames around the
pictures are dropped and the columns are sized to the round's longest word. At
`-n 3` a round needs roughly 60 columns of terminal; `-n 5` wants a wide window.

## How a round works

```
     1        2        3
     🐔       🌞       🍅

     a        b        c
   tomato    hen      sun
```

Answer `1b`, `2c`, `3a` in any order — and each answer can be typed either way
round, so `b1` is the same as `1b`. Input is forgiving: `1 A` and `1-a` work
too. `q` quits.

## Feedback

The game says almost nothing, on purpose.

- **Right:** the number and the letter both turn into a bold green ✔. That is
  the whole of the praise — no words, no exclamation marks.
- **Wrong:** the screen clears to a single 🙁 and stays there until the child
  presses a key. It does not name the pair they got wrong.
- **Neither:** a keypress that is not an answer at all — a stray letter, a
  number that is not on screen, a picture already ticked — draws no reaction.
  Only a real mismatch is treated as wrong.

"Presses a key" means any key, no ENTER. The game shells out to
`stty cbreak -echo` for that one moment and restores the terminal immediately
after (cbreak rather than raw, so Ctrl-C still works). Where that is not
possible — stdin is a pipe, no `stty`, Windows — it falls back to waiting for a
line, so the child presses a key and then ENTER.

The running ⭐ count in the header and the end-of-session summary are the only
other places the game keeps score.

## Adding pairs

Everything lives in `pairs.go`, whose doc comment carries the full reference:
where the words come from (hand-written for a small child, deliberately not the
official Unicode names — 🐔 is "hen" here, "chicken" to Unicode), the Unicode
data files to check a new emoji against, and a one-line `curl | awk` that lists
every emoji safe to use. Two rules the tests enforce:

- **Words are unique and lower-case** — a round identifies a pair by its word.
- **Emoji must render double-width.** Only use emoji that are colourful on
  their own. Anything needing a U+FE0F variation selector (`☀️` `✏️` `❄️` `✂️`)
  is one column wide in some terminals and two in others, which breaks the
  columns. `TestPairsAreWellFormed` fails if you add one.

Keep words to 8 letters or fewer: columns are sized to the round's longest word
and big mode doubles everything, so an 8-letter word needs 64 terminal columns
but an 11-letter one needs 82.

Rounds never repeat a pair used in the last 15 draws, and the three words in a
round always start with different letters — so a child who can only sound out
the first letter still has a way in. Both halves of the game get this from
`round.go`, so a pair added here turns up on the phone as well — after a
`make`, which rebuilds the wasm and rewrites `pairs.json` from the same slice.

## Tests

```sh
make check      # go vet, go test, build, and the two browser suites
```

`test/wasm.js` loads `docs/match.wasm` the way the page does and checks what
comes back; `test/player.js` does the same for the fallback generator, which it
slices out of the real `docs/app.js` rather than copying — there is no second
set of rules to drift out of step with the one the phone runs. Both go through
the same `checkRound`, so the two generators are held to one standard.

