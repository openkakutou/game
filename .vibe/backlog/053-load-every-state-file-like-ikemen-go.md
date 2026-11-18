---
status: todo
---
# Load Every State File Like Ikemen GO

## Description
The game builds a fighter from the `.def`'s `cns` file only (`native/internal/game/match.go`). It ignores `st`, `stN`, `stcommon` and every `.zss` file, so a Kung Fu Man match cannot start: "state 0 not found in loaded character", fidelity video of 2026-11-18. Load the states through engine's public loader (engine#057). `stcommon` is searched in the character folder, then in the game's `common/` folder (roadmap decision `031`, the counterpart of Ikemen GO's `data/`).

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO characters load as in Ikemen GO (roadmap decision `038`).

## Acceptance Criteria
- [ ] A `.cns` Kung Fu Man and a `.zss` Kung Fu Man, with their common1 in the character folder or in `common/`, start a match in the native game
- [ ] A character whose `stcommon` is found nowhere still loads its own states. The missing common file is reported, not fatal
- [ ] The web build loads states the same way, or the gap is recorded as an item
- [ ] Checked in the real game with the fidelity video, which is attached to the item's report

## Notes
Needs engine#057 released. Shipping files in `common/` stays game#038 (licences, effects, sounds).
