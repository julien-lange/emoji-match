'use strict';

// The player. A round is three (or two, or five) pictures down the left and the
// same words, shuffled differently, down the right; the child drags a line from
// one to the other. It knows nothing about how a round was made: it reads the
// wire format from wasm.go, whether that came back from the WebAssembly
// generator or from the small fallback below.
//
// The feedback is the terminal game's, moved to a screen. Right: a tick, a
// coloured line and a little burst of sparks. Wrong: a sad face and nothing
// else — it does not say which pair was wrong. Neither: a line let go over
// empty space, or over a card already ticked, draws no reaction at all, because
// that was never an answer.

const POOL_URL = 'pairs.json';
const STORE_KEY = 'match.progress.v1';

/* ------------------------------------------------------------- generator */

// The rounds come from match.wasm — the same Go generator the terminal game
// uses, so there is one curated pool and one set of rules about what makes a
// round. What follows is the fallback for the one case that matters: a 2.8 MB
// binary is a big thing to depend on absolutely, and a child tapping a home
// screen icon on a train should not be met with a blank page.
//
// It reads the same pool out of pairs.json and keeps to the same two rules that
// make a round playable — no pair seen in the last fifteen draws, and no two
// words starting with the same letter, so a child who can only sound out the
// first letter still has a way in. It also draws the very first round while the
// wasm is still arriving, which is why the game starts instantly.

const RECENT = 15;

let pool = [];        // {emoji, word}, from pairs.json
let recent = [];      // indices lately drawn, oldest first

function shuffle(a, rand) {
  for (let i = a.length - 1; i > 0; i--) {
    const j = Math.floor(rand() * (i + 1));
    [a[i], a[j]] = [a[j], a[i]];
  }
  return a;
}

function draw(n, avoidRecent, rand) {
  const order = shuffle([...pool.keys()], rand);
  const firsts = new Set();
  const idxs = [];
  for (const i of order) {
    if (idxs.length === n) break;
    if (avoidRecent && recent.includes(i)) continue;
    const f = pool[i].word[0];
    if (firsts.has(f)) continue;
    firsts.add(f);
    idxs.push(i);
  }
  if (idxs.length < n) return null;
  for (const i of idxs) {
    recent.push(i);
    if (recent.length > RECENT) recent.shift();
  }
  return idxs.map((i) => pool[i]);
}

function fallbackRound(n, rand = Math.random) {
  // Pool exhausted by the "recently seen" filter (only possible with a tiny
  // pool or a large n) — go round again allowing repeats.
  const chosen = draw(n, true, rand) || draw(n, false, rand);
  if (!chosen) return null;

  const pics = shuffle(chosen.slice(), rand);
  const words = chosen.slice();
  // Reshuffle while the answer is the giveaway first-to-first, second-to-second
  // layout, which teaches position rather than reading.
  for (let i = 0; i < 20; i++) {
    shuffle(words, rand);
    if (words.some((w, k) => w.word !== pics[k].word)) break;
  }

  return {
    n,
    emojis: pics.map((p) => p.emoji),
    words: words.map((p) => p.word),
    answer: pics.map((p) => words.findIndex((w) => w.word === p.word)),
  };
}

/* ----------------------------------------------------------------- state */

const stage = document.getElementById('stage');
const cols = document.getElementById('cols');
const leftCol = document.getElementById('left');
const rightCol = document.getElementById('right');
const ink = document.getElementById('ink');
const ctx = ink.getContext('2d');
const oops = document.getElementById('oops');
const cheer = document.getElementById('cheer');
const scoreEl = document.getElementById('score');
const infoEl = document.getElementById('info');
const nextBtn = document.getElementById('next');
const againBtn = document.getElementById('again');
const howMany = document.getElementById('howmany');

// A line for each matched pair, coloured by the word it lands on so two lines
// crossing are still two lines.
const HUES = ['#ff6b35', '#2ea44f', '#3b82f6', '#a855f7', '#f2b705'];
const hue = (j) => HUES[j % HUES.length];
const FACE = getComputedStyle(document.documentElement).getPropertyValue('--face').trim()
  || 'system-ui, sans-serif';

let round = null;         // {n, emojis, words, answer} — answer[i] is emoji i's word
let links = new Map();    // emoji index -> word index, correct matches only
let missed = new Set();   // emoji indices that have had a wrong try this round
let roundNo = 0;
let stars = 0;            // matched first time, across the whole session
let perRound = 3;

let picked = null;        // {side, i}: picked up by a tap, waiting for its partner
let drag = null;          // {side, i, x, y, moved}: a line being pulled out
let bad = null;           // the wrong line, held on screen under the sad face
let busy = false;         // sad face up: ignore the finger until it goes

let leftTiles = [], rightTiles = [];
let leftAt = [], rightAt = [];      // where a line starts and ends, in stage pixels
let leftBox = [], rightBox = [];    // the same cards as rectangles, for hit testing
let W = 0, H = 0, lw = 6, rowH = 60;

let wasmReady = false;
let stale = false;        // a newer app is waiting; go back for it at a safe moment
let hintUntil = 0, hinting = false, hintTimer = 0, cheerTimer = 0;

/* -------------------------------------------------------------- geometry */

// build lays out one round's cards. The DOM holds the cards, the canvas holds
// the lines, and fit below is what keeps the two agreeing about where anything
// is.
function build() {
  cols.style.setProperty('--n', round.n);
  leftCol.replaceChildren();
  rightCol.replaceChildren();
  leftTiles = round.emojis.map((e) => tile('pic', e, leftCol));
  rightTiles = round.words.map((w) => tile('word', w, rightCol));
}

function tile(kind, text, parent) {
  const el = document.createElement('div');
  el.className = 'tile ' + kind;
  el.textContent = text;
  parent.appendChild(el);
  return el;
}

// fit sizes the canvas and the type to whatever screen this is, then reads back
// where the cards actually landed. Both columns are fractions of the space, so
// the same code covers two pairs on a tablet and five on a small phone.
function fit() {
  if (!round) return;
  const dpr = Math.min(window.devicePixelRatio || 1, 3);
  W = stage.clientWidth;
  H = stage.clientHeight;
  ink.width = Math.round(W * dpr);
  ink.height = Math.round(H * dpr);
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

  const pic = leftTiles[0].getBoundingClientRect();
  const word = rightTiles[0].getBoundingClientRect();

  cols.style.setProperty('--pic-size', Math.floor(Math.min(pic.width, pic.height) * 0.72) + 'px');

  // One size for every word in the round, chosen so the longest of them fits —
  // the same reasoning as the terminal game sizing its columns to the round's
  // longest word. Measuring beats guessing at a character width: "elephant" and
  // "milk" are not eight and four times anything.
  ctx.font = '800 100px ' + FACE;
  let size = word.height * 0.58;
  const avail = word.width * 0.86;
  for (const w of round.words) size = Math.min(size, (100 * avail) / ctx.measureText(w).width);
  cols.style.setProperty('--word-size', Math.floor(size) + 'px');

  const box = stage.getBoundingClientRect();
  const at = (el, edge) => {
    const r = el.getBoundingClientRect();
    return { x: (edge === 'right' ? r.right : r.left) - box.left, y: r.top - box.top + r.height / 2 };
  };
  const rect = (el) => {
    const r = el.getBoundingClientRect();
    return { x: r.left - box.left, y: r.top - box.top, w: r.width, h: r.height };
  };
  // Lines leave a picture on its right and arrive at a word on its left, so
  // they only ever cross the gap between the columns.
  leftAt = leftTiles.map((t) => at(t, 'right'));
  rightAt = rightTiles.map((t) => at(t, 'left'));
  leftBox = leftTiles.map(rect);
  rightBox = rightTiles.map(rect);

  // How far apart the rows sit, which is not the same as how tall a card is
  // now that cards are capped and centred. Everything sized to the child's aim
  // — the weight of a line, and how near a finger has to land — follows from
  // the pitch, because that is what says how much room there is to miss by.
  rowH = round.n > 1 ? Math.abs(leftAt[1].y - leftAt[0].y) : pic.height;
  // Thick enough for a child to see it land, capped so a two-pair round on a
  // tablet does not draw ropes.
  lw = Math.max(5, Math.min(14, Math.round(rowH * 0.11)));
}

/* --------------------------------------------------------------- drawing */

function render() {
  if (!round) return;
  ctx.clearRect(0, 0, W, H);

  for (let i = 0; i < round.n; i++) {
    dot(leftAt[i], links.has(i) ? hue(links.get(i)) : 'rgba(47, 53, 66, .3)');
  }
  for (let j = 0; j < round.n; j++) {
    dot(rightAt[j], wordTaken(j) ? hue(j) : 'rgba(47, 53, 66, .3)');
  }

  // "Try dragging one of these" — rings that swell out of the pictures still
  // waiting, and only after the round has sat untouched for a while.
  const left = hintUntil - performance.now();
  if (left > 0) {
    const phase = ((1400 - left) / 700) % 1;
    for (let i = 0; i < round.n; i++) {
      if (links.has(i)) continue;
      ring(leftAt[i], lw * (1 + phase * 2.6), 'rgba(255, 107, 53,' + (1 - phase) * 0.75 + ')');
    }
  }

  for (const [i, j] of links) curve(leftAt[i], rightAt[j], 1, hue(j), lw);
  if (bad) curve(bad.a, bad.b, bad.dir, '#e5484d', lw);

  if (drag) {
    const from = drag.side === 'left' ? leftAt[drag.i] : rightAt[drag.i];
    const dir = drag.side === 'left' ? 1 : -1;
    curve(from, { x: drag.x, y: drag.y }, dir, '#ff6b35', lw);
    dot({ x: drag.x, y: drag.y }, '#ff6b35', lw * 0.85);
  }
}

// A gentle S rather than a straight line: two of them crossing stay legible,
// and it looks like something drawn rather than something computed. dir says
// which way the line sets off — pictures to the right, words to the left.
function curve(a, b, dir, colour, width) {
  const k = Math.max(22, Math.abs(b.x - a.x) * 0.45) * dir;
  ctx.strokeStyle = colour;
  ctx.lineWidth = width;
  ctx.lineCap = 'round';
  ctx.beginPath();
  ctx.moveTo(a.x, a.y);
  ctx.bezierCurveTo(a.x + k, a.y, b.x - k, b.y, b.x, b.y);
  ctx.stroke();
}

function dot(p, colour, r = lw * 0.7) {
  ctx.fillStyle = colour;
  ctx.beginPath();
  ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
  ctx.fill();
}

function ring(p, r, colour) {
  ctx.strokeStyle = colour;
  ctx.lineWidth = Math.max(2, lw * 0.4);
  ctx.beginPath();
  ctx.arc(p.x, p.y, r, 0, Math.PI * 2);
  ctx.stroke();
}

/* ---------------------------------------------------------- hit testing */

const wordTaken = (j) => [...links.values()].includes(j);
const taken = (side, i) => (side === 'left' ? links.has(i) : wordTaken(i));

// away is how far a point is from a card: zero anywhere on it, and the straight
// distance to its nearest edge otherwise.
function away(p, r) {
  const dx = Math.max(r.x - p.x, 0, p.x - (r.x + r.w));
  const dy = Math.max(r.y - p.y, 0, p.y - (r.y + r.h));
  return Math.hypot(dx, dy);
}

function nearestOn(p, side, reach) {
  const boxes = side === 'left' ? leftBox : rightBox;
  let best = -1, bestD = Infinity;
  for (let i = 0; i < boxes.length; i++) {
    const d = away(p, boxes[i]);
    if (d < bestD) { bestD = d; best = i; }
  }
  return bestD <= reach ? best : -1;
}

// Picking a line up is taken literally — a finger has to be on a card, or all
// but — because the gap between the columns is where lines live and a press
// there means nothing. Putting one down is forgiving: by then the child has
// said which card they set out from, and the only question left is which of a
// handful of cards on the far side they meant.
const GRAB = 14;
const reach = () => rowH * 0.75;

function nearest(p) {
  const l = nearestOn(p, 'left', GRAB);
  const r = nearestOn(p, 'right', GRAB);
  if (l < 0 && r < 0) return null;
  if (l < 0) return { side: 'right', i: r };
  if (r < 0) return { side: 'left', i: l };
  return away(p, leftBox[l]) <= away(p, rightBox[r]) ? { side: 'left', i: l } : { side: 'right', i: r };
}

/* -------------------------------------------------------------- pointers */

const local = (e, box) => ({ x: e.clientX - box.left, y: e.clientY - box.top });

function setPick(side, i) {
  clearPick();
  picked = { side, i };
  (side === 'left' ? leftTiles : rightTiles)[i].classList.add('picked');
}

function clearPick() {
  if (!picked) return;
  (picked.side === 'left' ? leftTiles : rightTiles)[picked.i].classList.remove('picked');
  picked = null;
}

stage.addEventListener('pointerdown', (e) => {
  if (busy || !round) return;
  quiet();

  const p = local(e, stage.getBoundingClientRect());
  const t = nearest(p);
  if (!t || taken(t.side, t.i)) { clearPick(); render(); return; }

  // Dragging is the game, but a small child often taps instead. A card tapped
  // and let go stays picked up, and a tap on the far side finishes the pair —
  // the same answer, reached the other way.
  if (picked && picked.side !== t.side) {
    const ei = picked.side === 'left' ? picked.i : t.i;
    const wi = picked.side === 'left' ? t.i : picked.i;
    clearPick();
    resolve(ei, wi);
    return;
  }

  drag = { side: t.side, i: t.i, x: p.x, y: p.y, x0: p.x, y0: p.y };
  setPick(t.side, t.i);
  stage.setPointerCapture(e.pointerId);
  render();
  e.preventDefault();
});

stage.addEventListener('pointermove', (e) => {
  if (!drag) return;
  const p = local(e, stage.getBoundingClientRect());
  drag.x = p.x;
  drag.y = p.y;
  render();
  e.preventDefault();
});

for (const type of ['pointerup', 'pointercancel']) {
  stage.addEventListener(type, (e) => {
    if (stage.hasPointerCapture(e.pointerId)) stage.releasePointerCapture(e.pointerId);
    const d = drag;
    drag = null;
    if (!d) return;

    const p = local(e, stage.getBoundingClientRect());

    // Let go where it was picked up: that was a tap, not a line, and the card
    // stays picked waiting for its partner. Judged by where the finger ended
    // rather than by whether any move arrived in between, because a finger
    // resting on a card trembles and reports moves that mean nothing.
    if (Math.hypot(p.x - d.x0, p.y - d.y0) < 12) { render(); return; }

    const other = d.side === 'left' ? 'right' : 'left';
    const j = nearestOn(p, other, reach());
    clearPick();

    // Let go over nothing, or over a card already ticked. That was not an
    // answer, so it draws no reaction — the line simply is not there any more.
    if (j < 0 || taken(other, j)) { render(); quiet(); return; }

    resolve(d.side === 'left' ? d.i : j, d.side === 'left' ? j : d.i);
  });
}

// The page must never scroll or zoom under the finger, on any browser.
stage.addEventListener('touchstart', (e) => e.preventDefault(), { passive: false });
document.addEventListener('gesturestart', (e) => e.preventDefault());

/* ------------------------------------------------------------- answering */

function resolve(ei, wi) {
  if (round.answer[ei] === wi) right(ei, wi);
  else wrong(ei, wi);
  quiet();
}

function right(ei, wi) {
  links.set(ei, wi);
  if (!missed.has(ei)) stars++;
  leftTiles[ei].classList.add('done');
  rightTiles[wi].classList.add('done');
  sparks(rightTiles[wi], round.emojis[ei]);
  againBtn.disabled = false;
  render();
  paint();
  save();
  if (links.size === round.n) celebrate();
}

// The whole of the negative feedback: a face, held up for a moment over the
// line that was wrong, and then both are gone. No words, and no clue as to
// which pair it should have been.
function wrong(ei, wi) {
  missed.add(ei);
  bad = { a: leftAt[ei], b: rightAt[wi], dir: 1 };
  busy = true;
  oops.hidden = false;
  shake(leftTiles[ei]);
  shake(rightTiles[wi]);
  render();
  setTimeout(() => {
    bad = null;
    busy = false;
    oops.hidden = true;
    render();
  }, 850);
}

function shake(el) {
  el.classList.remove('shake');
  void el.offsetWidth;            // restart the animation rather than ignore it
  el.classList.add('shake');
  el.addEventListener('animationend', () => el.classList.remove('shake'), { once: true });
}

/* ----------------------------------------------------------- celebration */

function celebrate() {
  document.getElementById('cheer-mark').textContent = '🎉';
  cheer.hidden = false;
  nextBtn.classList.add('ready');
  // Take the card away again so they can admire the lines they drew; the
  // glowing Next button is what carries on saying well done.
  clearTimeout(cheerTimer);
  cheerTimer = setTimeout(() => { cheer.hidden = true; }, 2200);
  confetti();
}

function confetti() {
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
  const glyphs = [...round.emojis, '🎉', '⭐'];
  for (let i = 0; i < 20; i++) {
    const bit = document.createElement('span');
    bit.className = 'confetti';
    bit.textContent = glyphs[i % glyphs.length];
    bit.style.left = Math.random() * 100 + '%';
    bit.style.animationDuration = 1.6 + Math.random() * 1.2 + 's';
    bit.style.animationDelay = Math.random() * 0.4 + 's';
    bit.addEventListener('animationend', () => bit.remove());
    stage.appendChild(bit);
  }
}

// One small burst out of the card just matched, so the reward lands where the
// child is already looking rather than somewhere they have to go and find.
function sparks(el, glyph) {
  if (matchMedia('(prefers-reduced-motion: reduce)').matches) return;
  const box = stage.getBoundingClientRect();
  const r = el.getBoundingClientRect();
  const x = r.left - box.left + r.width / 2;
  const y = r.top - box.top + r.height / 2;
  for (let i = 0; i < 7; i++) {
    const bit = document.createElement('span');
    bit.className = 'spark';
    bit.textContent = i % 2 ? '⭐' : glyph;
    bit.style.left = x + 'px';
    bit.style.top = y + 'px';
    const a = (i / 7) * Math.PI * 2 + Math.random();
    const far = 40 + Math.random() * 50;
    bit.style.setProperty('--dx', Math.cos(a) * far + 'px');
    bit.style.setProperty('--dy', Math.sin(a) * far + 'px');
    bit.addEventListener('animationend', () => bit.remove());
    stage.appendChild(bit);
  }
}

/* -------------------------------------------------------------- the hint */

// Nudging is only ever for a round that has been sitting untouched. Any touch
// at all puts it off again.
function quiet(after = 5000) {
  hintUntil = 0;
  clearTimeout(hintTimer);
  hintTimer = setTimeout(hint, after);
}

function hint() {
  if (busy || !round || links.size === round.n) return;
  clearTimeout(hintTimer);
  hintTimer = setTimeout(hint, 6000);
  hintUntil = performance.now() + 1400;
  if (hinting) return;
  hinting = true;
  const tick = () => {
    render();
    if (performance.now() < hintUntil) requestAnimationFrame(tick);
    else { hinting = false; render(); }
  };
  requestAnimationFrame(tick);
}

/* ------------------------------------------------------------ the rounds */

// A round comes from the wasm when it has arrived and from the fallback when it
// has not — which is how the very first round is on screen before a 2.8 MB
// binary has finished downloading.
function freshRound(n) {
  if (wasmReady && typeof window.emojiRound === 'function') {
    try {
      const d = JSON.parse(window.emojiRound(n));
      if (!d.error) return d;
    } catch (_) { /* fall through to the pool */ }
  }
  return pool.length ? fallbackRound(n) : null;
}

function nextRound() {
  // A newer app arrived while they were playing. Between two rounds is the
  // moment to take it: the score is already saved and there are no lines to
  // lose.
  if (stale) { location.reload(); return; }

  const d = freshRound(perRound);
  if (!d) return;

  round = d;
  roundNo++;
  links = new Map();
  missed = new Set();
  drag = null;
  bad = null;
  picked = null;
  busy = false;
  oops.hidden = true;
  clearTimeout(cheerTimer);
  cheer.hidden = true;
  nextBtn.classList.remove('ready');
  againBtn.disabled = true;

  build();
  fit();
  render();
  paint();
  save();
  quiet();
}

// again wipes the lines but not the memory of what went wrong: a pair matched
// only after a wrong try does not become a first-time star by starting over.
function again() {
  if (!round) return;
  links = new Map();
  picked = null;
  for (const t of [...leftTiles, ...rightTiles]) t.classList.remove('done', 'picked');
  againBtn.disabled = true;
  nextBtn.classList.remove('ready');
  cheer.hidden = true;
  render();
  paint();
  quiet();
}

function paint() {
  scoreEl.textContent = '⭐ ' + stars;
  infoEl.textContent = [
    '#' + roundNo,
    round ? round.n + ' pairs' : '—',
    wasmReady ? 'wasm' : 'json',
    // Hard spaces inside each item, ordinary ones between: on a narrow phone
    // the line then breaks between facts rather than through the middle of one.
  ].map((s) => s.replace(/ /g, ' ')).join(' · ');

  for (const b of howMany.children) b.setAttribute('aria-pressed', String(+b.textContent === perRound));
}

/* -------------------------------------------------------------- progress */

function save() {
  try {
    localStorage.setItem(STORE_KEY, JSON.stringify({ roundNo, stars, n: perRound }));
  } catch (_) { /* private browsing: not worth caring about */ }
}

function load() {
  try {
    return JSON.parse(localStorage.getItem(STORE_KEY)) || {};
  } catch (_) { return {}; }
}

/* ---------------------------------------------------------- fresh start */

// The way out of anything that has gone wrong for good: forget the score, throw
// away every file the browser is holding on our behalf, and come back for the
// whole app again. It asks first, because there is no undoing this one.
function wipe() {
  if (!confirm('Start again from scratch?\n\n'
             + 'This forgets the stars and downloads the app afresh.')) return;

  try { localStorage.removeItem(STORE_KEY); } catch (_) { /* nothing was saved anyway */ }

  // The cached copies, then the worker that serves them. Both are allowed to
  // fail: coming back for a fresh page is worth having either way, so the
  // reload happens whatever these two make of it.
  const jobs = [];
  if (typeof caches !== 'undefined') {
    jobs.push(caches.keys().then((keys) => Promise.all(keys.map((k) => caches.delete(k)))));
  }
  if (navigator.serviceWorker) {
    jobs.push(navigator.serviceWorker.getRegistration().then((reg) => reg && reg.unregister()));
  }
  const reload = () => location.reload();
  Promise.all(jobs.map((p) => p.catch(() => {}))).then(reload, reload);
}

/* -------------------------------------------------------------- controls */

nextBtn.addEventListener('click', nextRound);
againBtn.addEventListener('click', again);
document.getElementById('wipe').addEventListener('click', wipe);

for (const n of [2, 3, 4, 5]) {
  const b = document.createElement('button');
  b.textContent = String(n);
  b.setAttribute('aria-pressed', 'false');
  b.addEventListener('click', () => {
    if (perRound === n) return;
    perRound = n;
    roundNo--;              // changing the size is not a round played
    nextRound();
  });
  howMany.appendChild(b);
}

new ResizeObserver(() => {
  if (!round) return;
  fit();
  render();
}).observe(stage);

/* ----------------------------------------------------------------- wasm */

// The WebAssembly build is the same generator as the terminal game, so a round
// from it and a round from the fallback are indistinguishable by the time they
// get here.
function loadWasm() {
  if (typeof Go !== 'function') return;
  const go = new Go();
  const start = (result) => {
    go.run(result.instance);      // returns at the first block; the export is
    wasmReady = true;             // set before that happens
    if (!round) nextRound();      // pairs.json never turned up: this is the game
    else paint();                 // the footnote now says wasm
  };
  const bytes = () => fetch('match.wasm')
    .then((r) => r.arrayBuffer())
    .then((b) => WebAssembly.instantiate(b, go.importObject));

  if (WebAssembly.instantiateStreaming) {
    WebAssembly.instantiateStreaming(fetch('match.wasm'), go.importObject)
      .then(start)
      .catch(() => bytes().then(start).catch(() => {}));   // wrong MIME type, say
  } else {
    bytes().then(start).catch(() => {});
  }
}

/* ----------------------------------------------------------------- start */

const saved = load();
stars = saved.stars | 0;
roundNo = saved.roundNo | 0;
if ([2, 3, 4, 5].includes(saved.n)) perRound = saved.n;
paint();

// Whichever of the two arrives first starts the game.
fetch(POOL_URL)
  .then((r) => r.json())
  .then((data) => {
    pool = data.pairs;
    if (!round) nextRound();
  })
  .catch(() => {
    if (!round && !wasmReady) infoEl.textContent = 'could not load pairs.json';
  });

loadWasm();

// The worker serves the app from a cache, so a page can go on running the
// version it was launched with long after a new one is sitting there installed.
// A worker taking over is the signal that has happened, and the only way to see
// the new files is to go back for the page: at once if nothing has been matched
// yet, and otherwise at the next round, where there is nothing to lose by it.
if ('serviceWorker' in navigator && location.protocol === 'https:') {
  const updating = Boolean(navigator.serviceWorker.controller);  // not a first visit
  navigator.serviceWorker.addEventListener('controllerchange', () => {
    if (!updating) return;      // the first worker ever, claiming its first page
    stale = true;
    if (!links.size) location.reload();
  });
  navigator.serviceWorker.register('sw.js').catch(() => {});
}
