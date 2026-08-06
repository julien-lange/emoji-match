package main

// Pair is one emoji together with the word a child should read for it.
//
// # Where the words come from
//
// They are hand-written for a 4-5 year old, not taken from any dataset. About
// half of them differ from the emoji's official Unicode name on purpose,
// because the official names are written for search and screen readers rather
// than for a child sounding out letters:
//
//	🐶  we say "dog"    Unicode: "dog face"
//	🐔  we say "hen"    Unicode: "chicken"
//	🐳  we say "whale"  Unicode: "spouting whale"
//	⚽  we say "ball"   Unicode: "soccer ball"
//	🍉  we say "melon"  Unicode: "watermelon"
//
// British English where the two differ: 🔦 is "torch", not "flashlight".
//
// The rule is: one short concrete word a small child already says out loud, so
// the only new skill being practised is reading it. Avoid compound names
// ("ice cream"), plurals that fight the picture, and anything over 8 letters —
// see the note on width below.
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

// pairs is the curated pool every round is drawn from. Keep words unique — the
// round builder assumes it can identify a pair by its word.
var pairs = []Pair{
	// ---- animals ----
	{"🐶", "dog"},
	{"🐱", "cat"},
	{"🐷", "pig"},
	{"🐮", "cow"},
	{"🐭", "mouse"},
	{"🐔", "hen"},
	{"🐝", "bee"},
	{"🐟", "fish"},
	{"🐸", "frog"},
	{"🦆", "duck"},
	{"🐑", "sheep"},
	{"🐴", "horse"},
	{"🐰", "rabbit"},
	{"🦊", "fox"},
	{"🐻", "bear"},
	{"🦁", "lion"},
	{"🐯", "tiger"},
	{"🐘", "elephant"},
	{"🐍", "snake"},
	{"🐌", "snail"},
	{"🐛", "bug"},
	{"🐜", "ant"},
	{"🐧", "penguin"},
	{"🦉", "owl"},
	{"🐢", "turtle"},
	{"🐒", "monkey"},
	{"🦒", "giraffe"},
	{"🐳", "whale"},
	{"🦈", "shark"},
	{"🦓", "zebra"},
	{"🐐", "goat"},
	{"🦜", "parrot"},
	{"🐨", "koala"},
	{"🐼", "panda"},

	// ---- food ----
	{"🍎", "apple"},
	{"🍌", "banana"},
	{"🍇", "grapes"},
	{"🍊", "orange"},
	{"🍐", "pear"},
	{"🍋", "lemon"},
	{"🥕", "carrot"},
	{"🌽", "corn"},
	{"🍞", "bread"},
	{"🧀", "cheese"},
	{"🥚", "egg"},
	{"🍰", "cake"},
	{"🍪", "cookie"},
	{"🍕", "pizza"},
	{"🥛", "milk"},
	{"🍿", "popcorn"},
	{"🍩", "donut"},
	{"🥔", "potato"},
	{"🍅", "tomato"},
	{"🥦", "broccoli"},
	{"🍯", "honey"},
	{"🥜", "peanut"},
	{"🍉", "melon"},
	{"🍒", "cherry"},
	{"🧅", "onion"},

	// ---- things ----
	{"🚗", "car"},
	{"🚌", "bus"},
	{"🚲", "bike"},
	{"🚂", "train"},
	{"🚀", "rocket"},
	{"🚚", "truck"},
	{"🛴", "scooter"},
	{"⛵", "boat"},
	{"⚽", "ball"},
	{"🪑", "chair"},
	{"🧦", "sock"},
	{"👒", "hat"},
	{"👞", "shoe"},
	{"🧤", "glove"},
	{"🔑", "key"},
	{"📖", "book"},
	{"🎈", "balloon"},
	{"🎁", "gift"},
	{"🪁", "kite"},
	{"🧸", "teddy"},
	{"🥁", "drum"},
	{"🎸", "guitar"},
	{"🔔", "bell"},
	{"🔦", "torch"},
	{"🧹", "broom"},
	{"🪣", "bucket"},
	{"🥄", "spoon"},
	{"🧢", "cap"},
	{"👕", "shirt"},
	{"🧥", "coat"},
	{"👗", "dress"},
	{"🎒", "bag"},
	{"⌚", "watch"},
	{"📱", "phone"},
	{"💻", "laptop"},

	// ---- nature ----
	{"🌳", "tree"},
	{"🌸", "flower"},
	{"🍁", "leaf"},
	{"🌙", "moon"},
	{"⭐", "star"},
	{"🌞", "sun"},
	{"🌈", "rainbow"},
	{"🔥", "fire"},
	{"💧", "drop"},
	{"⛄", "snowman"},
	{"🌊", "wave"},
	{"🏠", "house"},
	{"🪨", "rock"},
	{"🐚", "shell"},
	{"🌵", "cactus"},
	{"🍄", "mushroom"},
	{"🌱", "plant"},
	{"🪵", "log"},

	// ---- body ----
	{"👀", "eyes"},
	{"👃", "nose"},
	{"👂", "ear"},
	{"👄", "lips"},
	{"🦶", "foot"},
	{"✋", "hand"},
	{"🦷", "tooth"},
	{"💛", "heart"},
}
