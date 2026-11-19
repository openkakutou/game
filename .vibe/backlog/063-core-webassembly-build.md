---
status: todo
depends_on: [062]
---
# Core WebAssembly build

## Description
Step 2 of item 060. A `cmd/core-wasm` entrypoint (like `cmd/mode-wasm`) exposing the core to JavaScript: load content from bytes, start a match or a mode, advance one tick with both players' inputs, read the state to draw as JSON. The web CI and deploy workflows build it with Go (setup-go) instead of downloading `engine`/`character`/`stage` WASM releases, so the web build always uses the same library versions as native (today it downloads engine v2.10.0 while native uses v2.23.0).

## Acceptance Criteria
- [ ] A smoke test under Node loads real content, runs ticks and reads the state, in CI
- [ ] The web build gets core.wasm from its own Go build, pinned by `native/go.mod`
- [ ] The tick API allocates no per-tick garbage beyond the state snapshot
