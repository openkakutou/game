import assert from "node:assert/strict";
// Smoke test of the game core's WebAssembly build, run under Node:
//   GOOS=js GOARCH=wasm go build -o /tmp/core.wasm ./cmd/core-wasm
//   node cmd/core-wasm/smoke.mjs /tmp/core.wasm "$(go env GOROOT)"
// It mounts the real test content, plays a match and reads the state.
import { readFileSync, readdirSync } from "node:fs";
import { createRequire } from "node:module";
import { join, relative } from "node:path";

const [wasmPath, goroot] = process.argv.slice(2);
createRequire(import.meta.url)(`${goroot}/lib/wasm/wasm_exec.js`);
const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(
  readFileSync(wasmPath),
  go.importObject,
);
go.run(instance);
const core = globalThis.OpenKakutouCore;

const contentRoot = new URL("../../testdata/content", import.meta.url).pathname;
function* files(dir) {
  for (const e of readdirSync(dir, { withFileTypes: true })) {
    const p = join(dir, e.name);
    if (e.isDirectory()) yield* files(p);
    else yield p;
  }
}
for (const p of files(contentRoot)) {
  const name = relative(contentRoot, p).split("\\").join("/");
  const r = JSON.parse(core.mount(name, new Uint8Array(readFileSync(p))));
  assert.equal(r.ok, true, `mount ${name}`);
}
// The fixture characters cannot walk: a narrow stage starts them in reach.
const arena = readFileSync(`${contentRoot}/stages/arena.def`, "utf8")
  .replace("leftbound = -160", "leftbound = -6")
  .replace("rightbound = 160", "rightbound = 6");
core.mount("stages/close.def", new TextEncoder().encode(arena));

const started = JSON.parse(
  core.startMatch(
    JSON.stringify({
      p1: "chars/fighter/fighter.def",
      p2: "chars/rival/rival.def",
      stage: "stages/close.def",
      setup: { rounds: 3, timeLimitSeconds: 60 },
      viewWidth: 320,
    }),
  ),
);
assert.equal(started.ok, true, JSON.stringify(started));
assert.equal(started.roundsToWin, 2);
assert.equal(started.fighters[1].health, 1000);

const press = JSON.stringify({ p1: { buttons: { a: true } }, p2: {} });
const release = JSON.stringify({ p1: {}, p2: {} });
let state;
let cues = 0;
for (let i = 0; i < 200; i++) {
  state = JSON.parse(core.tick(started.id, i % 2 === 0 ? press : release));
  assert.equal(state.ok, true, JSON.stringify(state));
  cues += state.cues.length;
}
assert.equal(state.tick, 200);
assert.ok(state.fighters[1].health < 1000, "P2 was hit");
assert.equal(state.fighters[0].health, 1000);
assert.equal(state.fighters[0].x, -6);
for (const f of state.fighters) {
  for (const k of [
    "x",
    "y",
    "facing",
    "anim",
    "animTime",
    "stateNo",
    "health",
    "power",
  ]) {
    assert.equal(typeof f[k], "number", k);
  }
}
assert.ok(cues > 0, "the attack played a sound");
assert.equal(state.round, 1);
assert.equal(typeof state.camX, "number");

assert.equal(JSON.parse(core.closeMatch(started.id)).ok, true);
assert.equal(JSON.parse(core.tick(started.id, release)).ok, false);
assert.equal(JSON.parse(core.mount(1)).ok, false);
assert.equal(JSON.parse(core.tick("x")).ok, false);
console.log("core-wasm smoke: ok");
process.exit(0);
