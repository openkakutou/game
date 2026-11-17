---
status: todo
depends_on: [031, 032]
---
# Quick versus as the first built-in mode

## Description
Roadmap decision `030`: rewrite quick versus as `modes/quick-versus.json` driving the built-in screens, with no behavior change for the player.

## Acceptance Criteria
- [ ] Quick versus plays exactly as before, now driven by its mode file
- [ ] The mode file ships in `modes/` (roadmap decision `031`)
- [ ] Existing match and visual regression tests stay green
