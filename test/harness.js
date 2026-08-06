'use strict';

// Shared plumbing. The point of both suites is to exercise the files that
// actually get published, not a copy of them, so everything here reads out of
// docs/ rather than reimplementing anything.

const fs = require('fs');
const path = require('path');

const DOCS = path.join(__dirname, '..', 'docs');

const read = (name) => fs.readFileSync(path.join(DOCS, name), 'utf8');
const readPool = () => JSON.parse(read('pairs.json')).pairs;

// loadLogic lifts the DOM-free half of app.js — the fallback generator — into a
// module. Slicing the real file rather than copying it is what keeps these
// tests honest: there is no second set of rules to drift out of step with the
// one the phone runs.
function loadLogic() {
  const src = read('app.js');
  const from = src.indexOf('const RECENT');
  const to = src.indexOf('/* ----------------------------------------------------------------- state */');
  if (from < 0 || to < 0) throw new Error('harness: section markers moved in app.js');

  const code = src.slice(from, to) +
    '\nmodule.exports = { fallbackRound,' +
    ' load: (p) => { pool = p; recent = []; },' +
    ' recent: () => recent };';

  const mod = new (require('module'))();
  mod._compile(code, path.join(DOCS, 'app-logic.js'));
  return mod.exports;
}

// A round is well formed if the two orders really are the same set of pairs and
// the answer really maps between them. Both generators are held to this, so it
// lives here rather than in either suite.
function checkRound(r, n, pool, fail, where) {
  if (!r) return fail(`${where}: no round at all`);
  if (r.n !== n || r.emojis.length !== n || r.words.length !== n || r.answer.length !== n) {
    return fail(`${where}: asked for ${n}, got ${r.n}/${r.emojis.length}/${r.words.length}/${r.answer.length}`);
  }

  const wordFor = new Map(pool.map((p) => [p.emoji, p.word]));
  const firsts = new Set();
  let identity = true;

  for (let i = 0; i < n; i++) {
    const want = wordFor.get(r.emojis[i]);
    if (want === undefined) fail(`${where}: ${r.emojis[i]} is not in the pool`);
    if (r.words[r.answer[i]] !== want) {
      fail(`${where}: ${r.emojis[i]} points at "${r.words[r.answer[i]]}", should be "${want}"`);
    }
    if (r.answer[i] !== i) identity = false;
    if (firsts.has(want[0])) fail(`${where}: two words start with "${want[0]}"`);
    firsts.add(want[0]);
  }

  if (new Set(r.words).size !== n) fail(`${where}: a word appears twice`);
  if (new Set(r.emojis).size !== n) fail(`${where}: an emoji appears twice`);
  if (new Set(r.answer).size !== n) fail(`${where}: two pictures point at one word`);
  // The giveaway layout teaches position rather than reading.
  if (identity) fail(`${where}: the answer is 1a/2b/3c`);
}

// results collects failures so both suites report and exit the same way.
function results(label) {
  let fails = 0;
  return {
    fail(msg) { console.log('FAIL:', msg); fails++; },
    get failed() { return fails; },
    done(...lines) {
      for (const line of lines) console.log(line);
      console.log(fails === 0 ? `${label} PASSED` : `${label}: ${fails} FAILURES`);
      process.exit(fails === 0 ? 0 : 1);
    },
  };
}

module.exports = { DOCS, read, readPool, loadLogic, checkRound, results };
