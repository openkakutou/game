---
status: done
depends_on: [031, 032]
---
# Versus as the first built-in mode

## Description
Roadmap decision `030`: rewrite today's two-player flow as the Versus mode, `modes/versus.jsonc` driving the built-in screens, with no behavior change for the player.

## Acceptance Criteria
- [ ] Versus plays exactly as today's flow, now driven by its mode file
- [ ] The mode file ships in `modes/` (roadmap decision `031`)
- [ ] Existing match and visual regression tests stay green

## Notes
- Once modes are read from `modes/` at runtime, the mode validator must refuse a flow with `results` before `match` (found reviewing 032: the results screen reads the match, which does not exist yet there).
- Versus' preselected time limit is now the rules' 99 s (was a hard-wired 60 s before 032).

## Done (2026-11-20)
Native reads `modes/versus.jsonc` (built-in fallback with a notice); the validator refuses `results` before `match` and a `characterSelect` without `sides`. The web follows through item 065; the release folder ships `modes/` with item 047.
