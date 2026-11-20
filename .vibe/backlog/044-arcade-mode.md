---
status: done
depends_on: [033, 036]
---
# Arcade mode

## Description
A built-in `modes/arcade.json`: one character, a ladder of CPU opponents with a final boss chosen through the roster filter, a continue screen, and a victory/ending screen.

## Acceptance Criteria
- [ ] Winning advances the ladder; losing offers continue
- [ ] The boss can be excluded from selection and still fought last
- [ ] Clearing the ladder shows the victory screen

## Done (2026-11-20)
Native (ADR 027): built-in Arcade (random ladder from the opponent pool, optional `final`, continue and victory screens from the pack, horizontal menus). Reviewed; ladder remapped on a roster re-sort. Continue/victory covered by tests, not by a smoke capture (no cheap way to lose under Xvfb). Web: through the core flow (065).
