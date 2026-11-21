---
status: todo
---
# Web match in the fighters' 320-wide space

## Description
Found by the audit of 2026-11-21 (high). Since 22a69c5 the match camera lives in the fighters' 320-wide space (`NewMatch`: `FighterScale`, `NewStageCamera(st, tension, k, mugenLocalWidth)`), and the native renderer converts with `VW / fighterWidth` (`gfx/screens.go`). The web core was not adapted: `native/internal/core/core.go:120,161` still converts `CameraX()` with `StageScale(st, viewWidth)` (1 when the canvas is the stage's localcoord width), and `core/tick.go:106-108` sends fighter positions unscaled. On a stage whose localcoord is not 320 wide (a 1280x720 one), the web background scrolls 4 times too little and the fighters stand ±70 px from the centre instead of ±280.

Decided: the web frames the match as native does. The core sends positions and `camX` in view pixels (`viewWidth / 320`) and the web scales fighter sprites by the same factor.

## Acceptance Criteria
- [ ] One helper in `game` gives the fighter-space-to-view ratio; `gfx` and `core` both use it
- [ ] On a 1280-wide stage, the web snapshot's `camX` and fighter `x`/`y` are in view pixels, and fighter sprites are drawn at `viewWidth / 320`
- [ ] A core test on a 1280-wide stage pins the starting positions and a camera move
- [ ] The "in stage units" comments on `Match.CameraX` (`match.go:660`) and `core.go:30` say what is true
- [ ] A 320-wide stage draws as before
