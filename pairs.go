package main

// Pair is one emoji together with the word a child should read for it.
//
// # Where the words come from
//
// The words come from the phonics scheme, not from the emoji. The children
// using this follow Little Wandle Letters and Sounds Revised (the UK DfE
// validated systematic synthetic phonics programme), so each stage is the set
// of words a child can decode from the letter-sounds taught so far — cat, bug,
// bag — and only then do we look for an emoji that shows it. A word with no
// good picture is dropped rather than the rule bent.
//
// Each Stage records which letters it assumes (Letters), and
// TestStageWordsAreDecodable fails if a word uses a letter outside them, so a
// stage can never quietly teach a sound that has not been taught yet.
//
// British English: 🔦 is "torch", not "flashlight".
//
// Keep words to 8 letters or fewer — see the note on width below.
//
// # Official sources
//
// Tricky words (words that cannot be fully sounded out, so they are not used
// here: there is no picture of "the") are the official Little Wandle Reception
// list, Wandle Learning Trust, June 2023:
//
//	Phase 2: and* as* be full go has* he her* his* I into is me no of pull
//	         push put she the to we            (*becomes decodable later)
//	Phase 3: all are by my pure sure they was you
//	Phase 4: come do have here like little love one out said says so some
//	         there today were what when
//
// The order in which letters are taught is Little Wandle's Phase 2 sequence
// (Reception, autumn 1). It was written down from knowledge of the programme,
// not read from the members-only planning grids, so check Letters against your
// school's copy before adding a stage.
//
// # Where the emoji come from — and how to check a new one
//
// The authority is the Unicode emoji data, not a font or a picker app:
//
//	https://unicode.org/Public/emoji/latest/emoji-test.txt      names + status
//	https://unicode.org/Public/UCD/latest/ucd/emoji/emoji-data.txt   properties
//	https://github.com/unicode-org/cldr/tree/main/common/annotations  CLDR names
//
// In CLDR, `<annotation cp="🐶" type="tts">dog face</annotation>` is the spoken
// name and the pipe-separated list on the untyped element is the keyword set;
// en_001.xml is the non-US English locale if you want British wording.
//
// Only emoji with the Emoji_Presentation property belong in this list. Those
// render as a full-colour, double-width glyph everywhere. Emoji that need a
// variation selector (U+FE0F) to turn colourful — ☀️ ✏️ ❄️ ✂️ — are one column
// wide in some terminals and two in others, so the columns stop lining up.
//
// The quick test: in emoji-test.txt, an emoji is safe if its fully-qualified
// form is a single code point. (Checked against emoji-data.txt: every
// single-code-point fully-qualified emoji has Emoji_Presentation, with no
// exceptions.) To list all 1184 candidates with their official names:
//
//	curl -s https://unicode.org/Public/emoji/latest/emoji-test.txt |
//	  awk -F' *; *' '/fully-qualified/ && split($1, cp, " ") == 1 {
//	    sub(/^[^#]*# */, ""); print }'
//
// TestPairsAreWellFormed fails if an added emoji is the wrong width, so a
// mistake here shows up as a failing test rather than a wonky screen.
//
// # A note on width
//
// Columns are sized to the round's longest word, and big mode doubles
// everything, so a three-picture round needs 2*(2+3*(longest+2)) terminal
// columns: 64 for an 8-letter word, but 82 for an 11-letter one like
// "caterpillar". Keep words to 8 letters and a round always fits an 80-column
// window.
type Pair struct {
	Emoji string
	Word  string
}

// Stage is one step of the phonics progression. Later stages add graphemes
// (sh, ch, ai, ee ...) and the words that use them.
type Stage struct {
	ID      int
	Name    string // short, for a button
	Detail  string // for the grown-up: what has been taught by this point
	Letters string // every letter a word in this stage may use
	Pairs   []Pair
}

// stages is the whole progression, in order. Only stage 1 exists so far. To
// add one, append here: the UIs, the JSON and the tests all pick it up.
var stages = []Stage{
	{
		ID:   1,
		Name: "Stage 1",
		Detail: "Little Wandle Reception autumn 1 (Phase 2): " +
			"s a t p, i n m d, g o c k, ck e u r, h b f l ff ll ss",
		Letters: "satpinmdgockeurhbfl",
		Pairs: []Pair{
			// short a
			{"🐱", "cat"},
			{"🎒", "bag"},
			{"🧢", "cap"},
			{"👒", "hat"},
			{"🐀", "rat"},
			{"🦇", "bat"},
			{"🐜", "ant"},
			// short e
			{"🐔", "hen"},
			{"🦵", "leg"},
			{"🧝", "elf"},
			{"🔔", "bell"},
			// short i
			{"🐷", "pig"},
			{"📌", "pin"},
			{"💋", "kiss"},
			// short o
			{"🐶", "dog"},
			{"🪵", "log"},
			{"🍲", "pot"},
			{"🧦", "sock"},
			// short u
			{"🐛", "bug"},
			{"🚌", "bus"},
			{"🌞", "sun"},
			{"☕", "cup"},
			{"🥜", "nut"},
			{"🦆", "duck"},
			{"🛁", "tub"},
			{"🤗", "hug"},
		},
	},
	{
		ID:   2,
		Name: "Stage 2",
		Detail: "Little Wandle Reception autumn 2 (Phase 2): " +
			"j v w x, y z zz qu, ch sh th ng nk",
		Letters: "satpinmdgockeurhbfljvwxyzq",
		Pairs: []Pair{
			// v w x y z
			{"🚐", "van"},
			{"🐺", "wolf"},
			{"🦊", "fox"},
			{"📦", "box"},
			{"🐂", "ox"},
			{"😋", "yum"},
			{"🤐", "zip"},
			{"⚡", "zap"},
			{"🐝", "buzz"},
			// ch
			{"🍟", "chip"},
			{"🐤", "chick"},
			{"💬", "chat"},
			// sh
			{"🚢", "ship"},
			{"🏪", "shop"},
			{"🐚", "shell"},
			{"🐟", "fish"},
			// th
			{"🛀", "bath"},
			// ng
			{"💍", "ring"},
			{"🎵", "song"},
			{"💥", "bang"},
			// nk
			{"🏦", "bank"},
			{"😉", "wink"},
			{"🔗", "link"},
		},
	},
}

// stageByID finds a stage, falling back to the first for an unknown id so a
// bad request still gets a playable round.
func stageByID(id int) *Stage {
	for i := range stages {
		if stages[i].ID == id {
			return &stages[i]
		}
	}
	return &stages[0]
}
