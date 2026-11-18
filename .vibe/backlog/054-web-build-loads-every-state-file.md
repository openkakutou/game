---
status: todo
---
# Web Build Loads Every State File

## Description
The native game now loads every state file of a character the way Ikemen GO does (item 053): `st`, `stN`, `stcommon`, `.cns` and `.zss`. The web build still sends engine only the `.def`'s `cns` file (`src/wasm/engine-bridge.ts`), so a Kung Fu Man match cannot start there either.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO characters load as in Ikemen GO (roadmap decision `038`).

## Acceptance Criteria
- [ ] The `character` WASM returns every state file of the `.def`, `.zss` included
- [ ] The app fetches `stcommon` from the character folder, then from a deployed `common/` folder
- [ ] The engine `newMatch` request carries `scripts` (engine `docs/wasm.md`), and the `engine.wasm` in use has that field
- [ ] A `.cns` Kung Fu Man and a `.zss` Kung Fu Man start a match in the web build, checked in the browser

## Notes
Gap found by item 053 (2026-11-18). Needs a roster manifest field and a `common/` deploy folder.
