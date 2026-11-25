---
status: done
---
# Tests for the HUD fill and the win icons

## Description
Found by the audit of 2026-11-21 (medium). `lifeFraction` (clamping, `lifeMax <= 0`), `winIcon` (win type to icon, default branch) and `winnerName` (out of range, nil character) in `native/internal/gfx/lifebar.go` (152, 239, 318) decide what the HUD shows and have no test.

## Acceptance Criteria
- [x] Table tests cover `lifeFraction` below 0, above the maximum and with a maximum of 0
- [x] `winIcon` is tested for each win type, the default included
- [x] `winnerName` is tested for a draw, an index out of range and a missing character

## Done
Table tests in `native/internal/gfx/lifebar_test.go`. The draw case found a bug: on a round without a winner the engine leaves `Winner` at its zero value, player 1, so `winnerName` named player 1. It now names a winner only on a KO or a time over, as the core's `roundWinner` already did for the web; it takes the fighters and the round result instead of the whole flow, so it can be tested.
