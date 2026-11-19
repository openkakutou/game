import assert from "node:assert/strict";
// Smoke test of the mode loader's WebAssembly build, run under Node:
//   GOOS=js GOARCH=wasm go build -o /tmp/mode.wasm ./cmd/mode-wasm
//   node cmd/mode-wasm/smoke.mjs /tmp/mode.wasm "$(go env GOROOT)"
import { readFileSync } from "node:fs";
import { createRequire } from "node:module";

const [wasmPath, goroot] = process.argv.slice(2);
createRequire(import.meta.url)(`${goroot}/lib/wasm/wasm_exec.js`);
const go = new globalThis.Go();
const { instance } = await WebAssembly.instantiate(
  readFileSync(wasmPath),
  go.importObject,
);
go.run(instance);

const example = readFileSync(
  new URL("../../internal/mode/versus.jsonc", import.meta.url),
  "utf8",
);
const ok = JSON.parse(
  globalThis.OpenKakutouMode.parse("versus.jsonc", example),
);
assert.equal(ok.ok, true);
assert.equal(ok.mode.id, "versus");
assert.equal(ok.mode.rules.rounds, 3);
assert.equal(ok.mode.flow[0].screen, "characterSelect");

const bad = JSON.parse(
  globalThis.OpenKakutouMode.parse(
    "bad.jsonc",
    example.replace('"rounds": 3', '"rounds": 4'),
  ),
);
assert.deepEqual(bad, {
  ok: false,
  error: {
    file: "bad.jsonc",
    field: "rules.rounds",
    reason: "must be odd (best of N)",
  },
});

const wrongArgs = JSON.parse(globalThis.OpenKakutouMode.parse(1));
assert.equal(wrongArgs.ok, false);
console.log("mode-wasm smoke: ok");
process.exit(0);
