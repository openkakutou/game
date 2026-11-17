---
status: todo
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
