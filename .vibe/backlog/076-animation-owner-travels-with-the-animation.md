---
status: todo
---
# The animation's owner travels with the animation

## Description
Found by the audit of 2026-11-21 (high). `Match.AnimNo`, `AnimTime` and `AnimOwner` are one value (`syncAnim` fills the three), but two copies drop the owner: the web snapshot (`core/tick.go:110-111`) and the afterimage `Ghost` (`afterimage.go:16-20`, recorded at `match.go:534`, drawn by `drawGhosts`). After a ChangeAnim2 (the victim of kfm's throw, state 820) the web looks the animation up in the thrown fighter's own character, and so do native afterimages: wrong sprites for the whole throw.

🔒 MUST: as Ikemen GO draws a ChangeAnim2 (roadmap decision `038`).

## Acceptance Criteria
- [ ] One value carries number, time and owner; `Ghost` records it and `drawGhosts` reads the owner's animations and sprites
- [ ] The web snapshot carries `animOwner` (validated by the bridge), and the web picks animations, sprite cache and SFF by that side
- [ ] A test on a real throw (kfm) pins the owner in the snapshot and in a recorded ghost
