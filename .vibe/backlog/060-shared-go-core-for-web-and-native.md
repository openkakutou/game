---
status: todo
---
# One Go core shared by the web and native builds

## Description
Roadmap decision `045`. Match flow, CPU opponent, camera, pause, roster/discovery rules, lifebar layout and audio logic exist today in both `src/` (TypeScript) and `native/internal/` (Go). Move everything that does not depend on the platform into one Go package tree in this repo, imported by the native build and compiled to WASM for the web build (like the mode loader of decision `030`). Each build keeps only its adapters: drawing, input devices, audio output, file access and storage.

Prioritized by roadmap decision `048`: no complete-game item (031+) starts before this one is done.

🔒 MUST: no behavior change — the native and web builds play the same match frame for frame before and after (differential tests, engine 036 / game 051, where available).

## Acceptance Criteria
- [ ] One Go implementation of match flow, CPU, camera, pause, roster rules, lifebar layout (via the `lifebar` library once released, roadmap `026`) and audio logic; the TypeScript copies are removed
- [ ] The web build runs that core through WASM; native imports it directly
- [ ] A test replays the same scripted match through both builds' core entry points and gets identical state
- [ ] `CLAUDE.md` and docs describe the core/adapters split
