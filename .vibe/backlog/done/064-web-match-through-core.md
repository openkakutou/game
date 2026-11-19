---
status: done
depends_on: [063]
---
# Web match through the core

## Description
Step 3 of item 060. The web match runs on core.wasm: it sends inputs each tick and draws the returned state. The TypeScript copies of match stepping, CPU opponent, camera, pause and HUD values are removed; rendering, input devices and audio output stay in TypeScript.

🔒 MUST: the web match plays the same as native for the same inputs.

## Acceptance Criteria
- [x] `src/cpu`, the camera and pause logic and the HUD value model are gone from TypeScript
- [x] The web build no longer loads `engine.wasm` (drawing data and the `character`/`stage` WASM move to the core in item 067)
- [x] Existing web e2e and visual tests pass, updated only where the old web behavior diverged from native
