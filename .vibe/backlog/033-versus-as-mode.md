---
status: todo
depends_on: [031, 032]
---
# Versus as the first built-in mode

## Description
Roadmap decision `030`: rewrite today's two-player flow as the Versus mode, `modes/versus.jsonc` driving the built-in screens, with no behavior change for the player.

## Acceptance Criteria
- [ ] Versus plays exactly as today's flow, now driven by its mode file
- [ ] The mode file ships in `modes/` (roadmap decision `031`)
- [ ] Existing match and visual regression tests stay green
