---
status: todo
---
# Author OpenKakutou's own common effects and sounds

## Description
Roadmap decision `037`: `fightfx.sff`/`.air` and `common.snd` have no verified license, and the Product Owner chose to author OpenKakutou's own instead of shipping them (2026-11-17). Produce hit sparks, guard sparks, super effects and common sounds under a free license (CC0 or CC BY, recorded in `common/LICENSES.md`), at the sprite groups, animation numbers and sound numbers real characters reference through the `F` prefix (fightfx) and common sound calls.

🔒 MUST: real MUGEN 1.0/1.1 and Ikemen GO characters that reference common effects and sounds find one at every number they use; the numbers come from a source (MUGEN docs, Ikemen GO source, real characters), not from memory.

## Acceptance Criteria
- [ ] Every common effect and sound number used by the acceptance characters and the corpus exists
- [ ] Assets are original work with their license recorded in `common/LICENSES.md`
- [ ] A real character's hit sparks and common sounds play from these files in a match
