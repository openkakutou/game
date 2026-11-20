---
status: todo
depends_on: [064]
---
# Web drawing data through the core

## Description
Step 3b of item 060 (split from 064). The web still loads `character.wasm` and `stage.wasm` for drawing data: sprite pixels of fighters and stages (`resolveSprites`, `resolveFighterSprites`, `resolveStageSprites`), stage animation frames and layers, stage boundaries, sprite flips, and roster/stage discovery (names). Expose what the web draws from the core (core.wasm: sprite pixels by sheet, stage layer state per tick, roster entries from the mounted content) and remove the two WASM downloads, so every web dependency comes from one Go build pinned by `native/go.mod`.

🔒 MUST: the web draws the same frames as before for the same match.

## Acceptance Criteria
- [ ] `src/wasm/bridge.ts` and `src/wasm/stage-bridge.ts` are gone; no `character`/`stage` WASM download remains in scripts and workflows
- [ ] Web visual tests pass, updated only where the old web drawing diverged from native

## Notes
- Item 030 (2026-11-20, ADR 025) draws every native screen through the cgo-free `native/internal/ui` draw list: the web can draw the same screens from it through the core.
