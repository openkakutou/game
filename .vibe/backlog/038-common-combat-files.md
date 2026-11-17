---
status: todo
depends_on: [037]
---
# Load common combat files from `common/`

## Description
Roadmap decision `031`: the common files MUGEN/Ikemen GO characters rely on live in `common/`: common states (`common1.cns`), common effects (`fightfx.sff`/`.air`), common sounds (`common.snd`). The game loads them from there and hands the common states to `engine`. Licensing per roadmap decision `037`: ship Ikemen GO's `common1.cns.zss` (MIT) with its notice in `common/LICENSE-ikemen-go.txt`; `fightfx` and `common.snd` ship only once a license is verified, otherwise OpenKakutou-authored replacements; every file's source and license listed in `common/LICENSES.md`.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO characters behave as with their engine's own common files.

Cross-repo: an unmodified real common1 needs engine#035.

## Acceptance Criteria
- [ ] Characters fall into the common get-hit, guard and lying-down states from `common/`
- [ ] Hit sparks and common sounds come from `common/`
- [ ] Replacing a file in `common/` changes the game accordingly
- [ ] `common/LICENSES.md` lists the source and license of every shipped file; no file ships without a verified license
