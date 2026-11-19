---
status: todo
depends_on: [060]
---
# Parse lifebars through the shared `lifebar` library

## Description
Roadmap decision `046`: lifebar parsing moves to `openkakutou/lifebar` (Go, WASM). Replace `native/internal/lifebar` (`ini.go`, `lifebar.go`, `layout.go`, `meter.go`) with the `lifebar` Go module in the shared core (item 060); `discover.go` (folder discovery and selection) stays in `game`. Item 059's remaining elements are then built in `lifebar` (its items 003-005), and drawn here.

Blocked until `lifebar` releases its WASM entrypoint (`lifebar` 007) with the features listed below; pin its exact tag (roadmap decisions `016`, `025`) and add this repo to `lifebar`'s release propagation.

🔒 MUST: no regression on any lifebar this repo loads today.

## Acceptance Criteria
- [ ] `native/internal/lifebar` keeps only discovery; parsing, layout and meter come from the `lifebar` module
- [ ] Existing lifebar tests pass, and the web build shows the same lifebar through the shared core
- [ ] Needs `lifebar` 001, 002 and `game` 060
