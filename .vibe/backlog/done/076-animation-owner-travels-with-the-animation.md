---
status: done
---
# The animation's owner travels with the animation

## Description
Found by the audit of 2026-11-21 (high). `Match.AnimNo`, `AnimTime` and `AnimOwner` are one value (`syncAnim` fills the three), but two copies drop the owner: the web snapshot (`core/tick.go:110-111`) and the afterimage `Ghost` (`afterimage.go:16-20`, recorded at `match.go:534`, drawn by `drawGhosts`). After a ChangeAnim2 (the victim of kfm's throw, state 820) the web looks the animation up in the thrown fighter's own character, and so do native afterimages: wrong sprites for the whole throw.

🔒 MUST: as Ikemen GO draws a ChangeAnim2 (roadmap decision `038`).

## Acceptance Criteria
- [x] One value carries number, time and owner; `Ghost` records it and `drawGhosts` reads the owner's animations and sprites
- [x] The web snapshot carries `animOwner` (validated by the bridge), and the web picks animations, sprite cache and SFF by that side
- [x] A test on a real throw (kfm) pins the owner in the snapshot and in a recorded ghost

## Done
2026-11-24. `game.Anim` (number, time, owner) replaces `Match.AnimNo` / `AnimTime` / `AnimOwner`; `Ghost` records it and `drawGhosts` draws from the owner's animations and sprites. The core snapshot carries `animOwner`; the bridge refuses a fighter without one of 0 or 1; the web resolves the frame, sprite cache and SFF and builds the draw command by that side. Tests: `TestMatch_AThrownKfmsAnimationAndGhostNameTheThrower` (a real kfm throw driven by input, the match's animation and the recorded ghost name P1), `TestSnapshot_CarriesTheSideEachAnimationIsDrawnFrom`, the bridge's animOwner test and the renderer's thrown-fighter test (red before the fix).
Left as is: the ghost drawing itself has no GL test harness; it reads the same owner as the fighter's draw.
