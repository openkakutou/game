---
status: todo
---
# Web comments that describe what is gone

## Description
Found by the audit of 2026-11-21 (medium). Several `src/` headers state things the code contradicts:
- `src/main.ts:281-286` (`showMatchSetup`) and `src/setup/setup-screen.ts:2-4`: the HUD and player input "aren't wired in yet" / "(not yet built)"; both are built and wired by `renderMatch`.
- `src/result/result-screen.ts:23`: "e.g. call `resetRound`"; no such function exists, the callback is wired to `nextRound`.
- `src/rendering/animation-resolution.ts:1-8` and `src/rendering/tick-scheduling.ts:1-3`: the data comes from "`engine`'s WASM response" (`animNo`) and `engine.tick()`; it comes from the core snapshot (`anim`, `animTime`) and the core's `tick`.
- `src/types/web-ui-kit.d.ts:1-5`: "installed package (v0.5.0)", while `package.json` pins `^0.23.1`.

## Acceptance Criteria
- [ ] Each comment says what the code does today, or is removed
- [ ] `web-ui-kit.d.ts` is deleted if the installed kit now ships its types
