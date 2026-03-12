// DVR State Tracker Tests
// Run: node web/dist/dvr_test.js

// Import the tracker from app.js — it's exposed on the IIFE for testing
const fs = require('fs');
const vm = require('vm');

// Minimal DOM stubs so app.js can load
const doc = {
  createElement: (tag) => ({
    tagName: tag.toUpperCase(),
    style: { cssText: '' },
    className: '',
    children: [],
    dataset: {},
    appendChild: function(c) { this.children.push(c); return c; },
    removeChild: function() {},
    addEventListener: function() {},
    removeEventListener: function() {},
    querySelector: function() { return null; },
    querySelectorAll: function() { return []; },
    setAttribute: function() {},
    getAttribute: function() { return null; },
    removeAttribute: function() {},
    replaceWith: function() {},
    insertBefore: function(n) { return n; },
    contains: function() { return false; },
    closest: function() { return null; },
    focus: function() {},
    remove: function() {},
    cloneNode: function() { return doc.createElement(tag); },
    getBoundingClientRect: function() { return { left: 0, right: 0, top: 0, bottom: 0, width: 0, height: 0 }; },
    get firstChild() { return this.children[0] || null; },
    get lastChild() { return this.children[this.children.length - 1] || null; },
    get nextSibling() { return null; },
    get parentNode() { return null; },
    set innerHTML(v) { this.children = []; this._html = v; },
    get innerHTML() { return this._html || ''; },
    set textContent(v) { this._text = v; },
    get textContent() { return this._text || ''; },
    set value(v) { this._value = v; },
    get value() { return this._value || ''; },
    set checked(v) { this._checked = v; },
    get checked() { return this._checked || false; },
    get offsetHeight() { return 0; },
    get offsetWidth() { return 0; },
    set onchange(fn) {},
    set onclick(fn) {},
    set oninput(fn) {},
  }),
  createTextNode: (text) => ({ nodeType: 3, textContent: text }),
  getElementById: (id) => doc.createElement('div'),
  body: { appendChild: function() {}, style: {} },
  head: { appendChild: function() {} },
  documentElement: { style: {} },
};

const context = {
  document: doc,
  window: {
    addEventListener: function() {},
    removeEventListener: function() {},
    location: { pathname: '/', hash: '', search: '' },
    history: { pushState: function() {}, replaceState: function() {} },
    innerWidth: 1024,
    innerHeight: 768,
    localStorage: {
      _data: {},
      getItem: function(k) { return this._data[k] || null; },
      setItem: function(k, v) { this._data[k] = v; },
      removeItem: function(k) { delete this._data[k]; },
    },
    matchMedia: () => ({ matches: false, addEventListener: function() {} }),
    getComputedStyle: () => ({ getPropertyValue: () => '' }),
  },
  localStorage: {
    _data: {},
    getItem: function(k) { return this._data[k] || null; },
    setItem: function(k, v) { this._data[k] = v; },
    removeItem: function(k) { delete this._data[k]; },
  },
  navigator: { userAgent: 'test' },
  console: console,
  setTimeout: setTimeout,
  clearTimeout: clearTimeout,
  setInterval: setInterval,
  clearInterval: clearInterval,
  fetch: () => Promise.resolve({ ok: true, json: () => Promise.resolve({}) }),
  HTMLElement: function() {},
  Event: function() {},
  CustomEvent: function() {},
};
context.window.document = doc;
context.self = context.window;

// Load app.js in sandbox to extract createDVRTracker
const appCode = fs.readFileSync(__dirname + '/app.js', 'utf8');
const script = new vm.Script(appCode, { filename: 'app.js' });
try { script.runInNewContext(context); } catch(e) { /* init() DOM crash expected */ }

const createDVRTracker = context.window._testExports && context.window._testExports.createDVRTracker;
if (!createDVRTracker) {
  console.error('FAIL: createDVRTracker not exported. Add window._testExports = { createDVRTracker } to app.js');
  process.exit(1);
}

// Test framework
let passed = 0;
let failed = 0;

function assert(condition, msg) {
  if (!condition) {
    console.error('  FAIL: ' + msg);
    failed++;
  } else {
    passed++;
  }
}

function assertClose(actual, expected, tolerance, msg) {
  if (Math.abs(actual - expected) > tolerance) {
    console.error('  FAIL: ' + msg + ' (got ' + actual + ', expected ' + expected + ')');
    failed++;
  } else {
    passed++;
  }
}

function test(name, fn) {
  console.log('  ' + name);
  fn();
}

// ═══════════════════════════════════════════════
// Tests
// ═══════════════════════════════════════════════

console.log('\n=== DVR Tracker Tests ===\n');

test('Initial state', () => {
  const t = createDVRTracker(true);
  assert(t.getPos(0) === 0, 'pos should be 0 initially');
  assert(t.getBuffered() === 0, 'buffered should be 0 initially');
  assert(t.isSeeking() === false, 'should not be seeking initially');
  const d = t.getDisplay(0);
  assert(d.pos === 0, 'display pos should be 0');
  assert(d.total === 0, 'display total should be 0');
  assert(d.pct === 0, 'display pct should be 0');
});

test('Normal playback — pos grows with video.currentTime', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(10);
  assert(t.getPos(5) === 5, 'pos = 0 + 5 = 5');
  const d = t.getDisplay(5);
  assert(d.pos === 5, 'display pos = 5');
  assert(d.total === 10, 'display total = buffered = 10');
  assert(d.pct === 50, 'display pct = 50%');
});

test('Buffered grows over time', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(5);
  assert(t.getDisplay(3).pct === 60, '3/5 = 60%');
  t.updateBuffered(10);
  assert(t.getDisplay(3).pct === 30, '3/10 = 30%');
  t.updateBuffered(100);
  assert(t.getDisplay(50).pct === 50, '50/100 = 50%');
});

test('Single stall + auto-seek', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(35);

  // Playing at video.currentTime=30 when stall occurs
  const seekTime = t.startSeek(30);
  assert(seekTime === 30, 'should seek to pos 30');
  assert(t.isSeeking() === true, 'should be seeking');

  // Seek completes, video.currentTime resets to 0
  t.completeSeek();
  assert(t.isSeeking() === false, 'should no longer be seeking');
  assert(t.getPos(0) === 30, 'pos = 30 + 0 = 30');

  // Video plays for 5 more seconds
  t.updateBuffered(40);
  assert(t.getPos(5) === 35, 'pos = 30 + 5 = 35');
  const d = t.getDisplay(5);
  assert(d.pos === 35, 'display pos = 35');
  assert(d.total === 40, 'display total = 40');
  assertClose(d.pct, 87.5, 0.1, 'display pct = 87.5%');
});

test('Multiple stalls — no offset accumulation bug', () => {
  const t = createDVRTracker(true);

  // First stall at video.currentTime=60, buffered=65
  t.updateBuffered(65);
  const seek1 = t.startSeek(60);
  assert(seek1 === 60, 'first seek to 60');
  t.completeSeek();
  assert(t.getPos(0) === 60, 'after first seek, pos = 60');

  // Video plays for 10s, second stall at video.currentTime=10
  t.updateBuffered(80);
  const seek2 = t.startSeek(10);
  assert(seek2 === 70, 'second seek to 60+10=70');
  t.completeSeek();
  assert(t.getPos(0) === 70, 'after second seek, pos = 70');

  // Video plays for 5s, third stall
  t.updateBuffered(90);
  const seek3 = t.startSeek(5);
  assert(seek3 === 75, 'third seek to 70+5=75');
  t.completeSeek();
  assert(t.getPos(0) === 75, 'after third seek, pos = 75');
  assert(t.getPos(10) === 85, 'playing for 10s: pos = 85');
});

test('Seek capped at buffered', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(50);

  // Video.currentTime somehow exceeds buffered (shouldn't normally happen)
  const seekTime = t.startSeek(60);
  assert(seekTime === 50, 'seek capped at buffered (50)');
  t.completeSeek();
  assert(t.getPos(0) === 50, 'pos after capped seek = 50');
});

test('Seek rejected when already seeking', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(50);

  const seek1 = t.startSeek(30);
  assert(seek1 === 30, 'first seek accepted');

  const seek2 = t.startSeek(40);
  assert(seek2 === null, 'second seek rejected (already seeking)');
});

test('Seek rejected when buffered is 0', () => {
  const t = createDVRTracker(true);
  // buffered is 0
  const seekTime = t.startSeek(10);
  assert(seekTime === null, 'seek rejected when buffered=0');
});

test('Display pos capped at total for live streams', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(10);

  // pos (15) > buffered (10) — e.g. status fetch lagging
  const d = t.getDisplay(15);
  assert(d.pos === 15, 'raw pos should be 15');
  assert(d.total === 10, 'total should be 10');
  assert(d.pct === 100, 'pct capped at 100');
});

test('Non-live (recorded) uses duration as total', () => {
  const t = createDVRTracker(false, 120);
  t.updateBuffered(60);

  const d = t.getDisplay(30);
  assert(d.pos === 30, 'pos = 30');
  assert(d.total === 120, 'total = duration (120), not buffered');
  assert(d.pct === 25, 'pct = 30/120 = 25%');
});

test('Reset returns to live edge', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(100);

  // Seek to position 50
  t.startSeek(50);
  t.completeSeek();
  assert(t.getPos(0) === 50, 'pos after seek = 50');
  assert(t.getSeekOffset() === 50, 'seekOffset = 50');

  // Reset to live
  t.reset();
  assert(t.getPos(0) === 0, 'pos after reset = 0');
  assert(t.getSeekOffset() === 0, 'seekOffset after reset = 0');
  assert(t.isSeeking() === false, 'not seeking after reset');
});

test('Manual seek to specific position', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(100);

  // User clicks seek bar at position 40
  const seekTime = t.seekTo(40);
  assert(seekTime === 40, 'manual seek to 40');
  t.completeSeek();
  assert(t.getPos(0) === 40, 'pos = 40');
  assert(t.getPos(10) === 50, 'pos after 10s = 50');
});

test('Manual seek rejected beyond buffered', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(50);

  const seekTime = t.seekTo(60);
  assert(seekTime === null, 'seek beyond buffered rejected');
});

test('Stall during playback resumes from correct position', () => {
  const t = createDVRTracker(true);

  // Simulate: play 30s, buffered=35, stall
  t.updateBuffered(35);
  const seek1 = t.startSeek(30); // video.currentTime=30
  assert(seek1 === 30, 'seek to 30');
  t.completeSeek();

  // Play 3 more seconds from new source
  // buffered is now 40
  t.updateBuffered(40);
  assert(t.getPos(3) === 33, 'pos = 30+3 = 33');

  // Another stall at video.currentTime=3
  const seek2 = t.startSeek(3); // video.currentTime=3
  assert(seek2 === 33, 'seek to 30+3=33');
  t.completeSeek();

  t.updateBuffered(45);
  assert(t.getPos(0) === 33, 'pos immediately after second seek = 33');
  assert(t.getPos(7) === 40, 'pos after 7s = 40');
});

test('getBufferDisplay for live without EPG', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(100);

  const d = t.getDisplay(50);
  assert(d.pos === 50, 'pos = 50');
  assert(d.total === 100, 'total = buffered = 100');
  assert(d.pct === 50, 'pct = 50%');

  // After seek to 80
  t.seekTo(80);
  t.completeSeek();
  t.updateBuffered(110);

  const d2 = t.getDisplay(5);
  assert(d2.pos === 85, 'pos = 80+5 = 85');
  assert(d2.total === 110, 'total = 110');
  assertClose(d2.pct, 77.3, 0.1, 'pct ≈ 77.3%');
});

test('Concurrent state consistency', () => {
  const t = createDVRTracker(true);
  t.updateBuffered(100);

  // Seek, then immediately update buffered (simulating concurrent operations)
  t.seekTo(50);
  t.updateBuffered(105);
  t.completeSeek();
  t.updateBuffered(110);

  const d = t.getDisplay(5);
  assert(d.pos === 55, 'pos = 50+5 = 55');
  assert(d.total === 110, 'total reflects latest buffered');
});

// ═══════════════════════════════════════════════
// Summary
// ═══════════════════════════════════════════════

console.log('\n' + (failed === 0 ? 'All ' + passed + ' assertions passed.' : failed + ' FAILED, ' + passed + ' passed.'));
if (failed > 0) process.exit(1);
