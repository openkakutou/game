---
status: todo
---
# Tests for the HUD fill and the win icons

## Description
Found by the audit of 2026-11-21 (medium). `lifeFraction` (clamping, `lifeMax <= 0`), `winIcon` (win type to icon, default branch) and `winnerName` (out of range, nil character) in `native/internal/gfx/lifebar.go` (152, 239, 318) decide what the HUD shows and have no test.

## Acceptance Criteria
- [ ] Table tests cover `lifeFraction` below 0, above the maximum and with a maximum of 0
- [ ] `winIcon` is tested for each win type, the default included
- [ ] `winnerName` is tested for a draw, an index out of range and a missing character
