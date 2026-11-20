---
status: done
depends_on: [033, 043]
---
# Training mode

## Description
A built-in `modes/training.json`: one player against a dummy, unlimited round time and life refill, with pause entries to reset positions and set the dummy (stand, crouch, jump, guard, CPU).

If `engine` lacks a capability (life refill, dummy guard), split an `engine` item first.

## Acceptance Criteria
- [ ] Life refills and the round never ends
- [ ] Reset positions and dummy settings work from the pause menu
- [ ] The move list is reachable from the pause menu

## Done (2026-11-20)
Native: built-in Training (`rules.training`), life refill after 60 quiet ticks or at 0, no time limit, dummy Stand/Crouch/Jump/Guard/CPU. Guard holds back only: real blocking waits for engine item 023. Reviewed, 1 finding fixed. Web: through the core flow (065).
