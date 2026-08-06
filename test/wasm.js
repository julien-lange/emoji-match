'use strict';

// Loads docs/match.wasm the way the page does and checks that what comes back
// is a well-formed round in the wire format the player reads — so the two
// halves of the app cannot drift apart.

const fs = require('fs');
const path = require('path');
const { DOCS, readPool, checkRound, results } = require('./harness');

require(path.join(DOCS, 'wasm_exec.js'));   // defines globalThis.Go

const pool = readPool();
const r = results('WASM CHECKS');

const go = new Go();
WebAssembly.instantiate(fs.readFileSync(path.join(DOCS, 'match.wasm')), go.importObject).then((res) => {
  go.run(res.instance);   // returns at the first block; the export is set before that

  if (typeof globalThis.emojiRound !== 'function') {
    console.log('FAIL: emojiRound was not exported');
    process.exit(1);
  }

  const t0 = Date.now();
  let made = 0;

  for (const n of [2, 3, 4, 5]) {
    for (let s = 0; s < 40; s++) {
      const d = JSON.parse(globalThis.emojiRound(n, 1000 + s));
      if (d.error) { r.fail(d.error); continue; }
      checkRound(d, n, pool, r.fail.bind(r), `${n} pairs, seed ${1000 + s}`);
      made++;
    }
  }

  // Out of range is clamped rather than refused: the page should never be able
  // to ask for a round the child cannot play.
  for (const [asked, want] of [[0, 2], [1, 2], [9, 5]]) {
    const d = JSON.parse(globalThis.emojiRound(asked, 7));
    if (d.n !== want) r.fail(`asked for ${asked} pairs, got ${d.n}, want ${want}`);
  }
  if (JSON.parse(globalThis.emojiRound(undefined, 7)).n !== 3) r.fail('no argument should mean three pairs');

  // Determinism: a seed is for the tests, so it has to mean something.
  const a = globalThis.emojiRound(3, 42);
  if (a !== globalThis.emojiRound(3, 42)) r.fail('the same seed gave different rounds');
  if (a === globalThis.emojiRound(3, 43)) r.fail('different seeds gave the same round');

  // The unseeded path is the one the page uses, and its picker has to remember
  // what it has already dealt: no pair twice in fifteen draws.
  const seen = [];
  let repeats = 0;
  for (let i = 0; i < 60; i++) {
    const d = JSON.parse(globalThis.emojiRound(3));
    checkRound(d, 3, pool, r.fail.bind(r), `unseeded #${i}`);
    for (const w of d.words) {
      if (seen.slice(-15).includes(w)) repeats++;
      seen.push(w);
    }
  }
  if (repeats) r.fail(`${repeats} words came round again inside fifteen draws`);

  r.done(
    `${made} seeded rounds + 60 from the session picker in ${Date.now() - t0}ms`,
    `pool: ${pool.length} pairs`,
  );
});
