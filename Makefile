# Everything the browser needs lives in docs/, which is what GitHub Pages
# serves. `make` rebuilds both halves of it.

PORT ?= 8732
CHROME ?= /Applications/Google Chrome.app/Contents/MacOS/Google Chrome

.PHONY: all wasm pairs icons serve check clean

all: wasm pairs

# The same generator as the terminal game, compiled for the browser. This is
# where the rounds come from: nothing is baked ahead of time, so the game never
# runs out.
wasm:
	GOOS=js GOARCH=wasm go build -o docs/match.wasm .
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" docs/wasm_exec.js

# The pool as plain JSON, beside the wasm. The page draws its very first round
# from this while the binary is still downloading, and falls back to it for good
# if the binary never arrives.
pairs:
	go run . -pairs docs/pairs.json

# Only needed if icon.svg changes. Needs ImageMagick, which is why the results
# are committed rather than built on demand.
icons:
	magick -background none docs/icon.svg -resize 512x512 -depth 8 -strip docs/icon-512.png
	magick -background none docs/icon.svg -resize 180x180 -depth 8 -strip docs/icon-180.png

serve:
	@echo "http://localhost:$(PORT)/"
	cd docs && python3 -m http.server $(PORT)

# The suites read what is in docs/, so rebuild both halves first — otherwise a
# changed pool would be checked against yesterday's binary.
check: wasm pairs
	go vet ./...
	go test ./...
	go build -o /dev/null .
	node test/wasm.js
	node test/player.js

clean:
	rm -f docs/match.wasm docs/wasm_exec.js docs/pairs.json
