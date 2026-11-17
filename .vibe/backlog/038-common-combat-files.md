---
status: todo
depends_on: [037]
---
# Load common combat files from `common/`

## Description
Roadmap decision `031`: the common files MUGEN/Ikemen GO characters rely on live in `common/`: common states (`common1.cns`), common effects (`fightfx.sff`/`.air`), common sounds (`common.snd`). The game loads them from there and hands the common states to `engine`. Ship OpenKakutou-authored versions (Elecbyte's and Ikemen GO's files are not ours to redistribute without checking their license), compatible with what real characters expect.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO characters behave as with their engine's own common files.

Cross-repo: an unmodified real common1 needs engine#035.

## Acceptance Criteria
- [ ] Characters fall into the common get-hit, guard and lying-down states from `common/`
- [ ] Hit sparks and common sounds come from `common/`
- [ ] Replacing a file in `common/` changes the game accordingly
- [ ] Licenses of shipped common files recorded
