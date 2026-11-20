---
status: done
depends_on: [033]
---
# Watch mode (CPU vs CPU)

## Description
A built-in `modes/watch.json`: the player picks two characters and a stage, both fighters are CPU-driven.

## Acceptance Criteria
- [ ] Both sides are controlled by the CPU
- [ ] The match can be paused and left like any other

## Done (2026-11-20)
Native: built-in `watch.jsonc` (both sides CPU), built-in modes generalized (each falls back to its embedded copy). Web: through the core flow (065).
