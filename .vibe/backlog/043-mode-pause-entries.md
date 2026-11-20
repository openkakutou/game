---
status: done
depends_on: [021, 031]
---
# Pause menu entries provided by the mode

## Description
Roadmap decision `030`: the pause menu lists the game's entries (resume, restart, options, move list, back to select, quit) plus entries a mode declares from a fixed set the game offers (e.g. reset positions, dummy settings for training).

## Acceptance Criteria
- [ ] A mode can add entries only from the offered set; an unknown entry is a validation error
- [ ] Versus shows only the game's entries

## Done (2026-11-20)
Native: mode field `pause` with `resetPositions` and `dummy`, inserted after Restart. Reset is ignored during a round-end banner and keeps the round being played. Web: through the core flow (065).
