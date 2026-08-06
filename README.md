# Emoji Match

A tiny terminal game for 4–5 year olds. Each round shows three pictures and
three words, shuffled independently. The child types a number and a letter —
`1a`, or `a1`, whichever order they reach for — then presses ENTER, so they
practise reading *and* find their way around a keyboard.

Everything is drawn at double size by default, emoji included.

## Run it

```sh
go run .          # or: go build -o emoji-match . && ./emoji-match
```

## Options

| Flag | Meaning |
| --- | --- |
| `-n 3` | pairs per round, 2–5 (start at `-n 2` for a first session) |
| `-big` | double-size text and emoji, on by default; `-big=false` for normal |
| `-rounds 0` | stop after N rounds; 0 plays until you type `q` |
| `-nocolor` | plain text (also honours `NO_COLOR`) |
| `-seed 0` | fix the randomness, handy for testing |

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
the first letter still has a way in.

```sh
go test ./...
```

