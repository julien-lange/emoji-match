'use strict';

// Checks the fallback generator inside docs/app.js — the one that runs when
// match.wasm has not arrived, and the one that draws the very first round of
// every session while the binary is still downloading. It has to keep to the
// same rules as the Go generator it stands in for, so it is held to the same
// checks the wasm suite uses.

const { readPool, loadLogic, checkRound, results } = require('./harness');

const pool = readPool();
const L = loadLogic();
const r = results('PLAYER CHECKS');

// The pool the page reads must be the pool the tests believe in.
if (!pool.length) r.fail('pairs.json is empty');
const words = new Set();
for (const p of pool) {
  if (!/^[a-z]+$/.test(p.word)) r.fail(`"${p.word}" is not a single lower-case word`);
  if (words.has(p.word)) r.fail(`duplicate word "${p.word}"`);
  words.add(p.word);
  if ([...p.emoji].length !== 1) r.fail(`"${p.emoji}" (${p.word}) is not a single code point`);
}

// A seeded generator, so a failure here can be reproduced. Anything the page
// itself does still goes through Math.random.
function seeded(s) {
  let x = s >>> 0;
  return () => {
    x ^= x << 13; x >>>= 0;
    x ^= x >> 17;
    x ^= x << 5; x >>>= 0;
    return x / 4294967296;
  };
}

L.load(pool);
const rand = seeded(12345);

for (const n of [2, 3, 4, 5]) {
  for (let i = 0; i < 40; i++) {
    checkRound(L.fallbackRound(n, rand), n, pool, r.fail.bind(r), `fallback ${n} pairs #${i}`);
  }
}

// The recently-seen memory, which is what stops 🐶 coming back two rounds later.
L.load(pool);
const seen = [];
let repeats = 0;
for (let i = 0; i < 80; i++) {
  const d = L.fallbackRound(3, rand);
  for (const w of d.words) {
    if (seen.slice(-15).includes(w)) repeats++;
    seen.push(w);
  }
}
if (repeats) r.fail(`${repeats} words came round again inside fifteen draws`);
if (L.recent().length !== 15) r.fail(`the memory holds ${L.recent().length} pairs, want 15`);

// The pool is far bigger than one round, so a run of rounds should reach most
// of it rather than circling a favourite corner.
L.load(pool);
const used = new Set();
for (let i = 0; i < 200; i++) for (const w of L.fallbackRound(3, rand).words) used.add(w);
if (used.size < pool.length * 0.9) {
  r.fail(`600 draws only ever used ${used.size} of ${pool.length} pairs`);
}

r.done(`${pool.length} pairs in the pool; 200 fallback rounds reached ${used.size} of them`);
