---
status: done
depends_on: [037, 050]
---
# Load common combat files from `common/`

## Description
Roadmap decision `031`: the common files MUGEN/Ikemen GO characters rely on live in `common/`: common states (`common1.cns`), common effects (`fightfx.sff`/`.air`), common sounds (`common.snd`). The game loads them from there and hands the common states to `engine`. Licensing per roadmap decision `037`: ship Ikemen GO's `common1.cns.zss` (MIT) with its notice in `common/LICENSE-ikemen-go.txt`; `fightfx` and `common.snd` ship only once a license is verified, otherwise OpenKakutou-authored replacements; every file's source and license listed in `common/LICENSES.md`.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO characters behave as with their engine's own common files.

Cross-repo: an unmodified real common1 needs engine#035.

## Acceptance Criteria
- [x] Characters fall into the common get-hit, guard and lying-down states from `common/`
- [x] Hit sparks and common sounds come from `common/`
- [x] Replacing a file in `common/` changes the game accordingly
- [x] `common/LICENSES.md` lists the source and license of every shipped file; no file ships without a verified license

## Notes
- 050 done (2026-11-20): `common/` holds the generated fightfx and common.snd. This item also checks 050's last criterion: a real character's hit sparks and common sounds play from them in a match (needs engine Explod/spark support, item 024 / engine#029, for the sparks to show).

## Blocked
2026-11-20 — Shipped: Ikemen GO's `common1.cns.zss` (MIT, pinned 54d4148a) with its notice, picked up by `stcommon = common1.cns`; F-prefixed `PlaySnd` reads `common/common.snd` in the native build; a test keeps `common/LICENSES.md` complete. Left: get-hit, guard and lying-down through the unmodified common1 need engine#035; hit sparks need item 024 (engine Explod/sparks); HitDef `hitsound`/`guardsound` need the engine to emit them as sound events. Channels and `lowPriority` stay in item 055.

## Done
2026-11-21 — Engine v2.36.0 runs both real common1 files through get-hit, guard, fall, lying down and getting up (engine items 023, 035, 049-052) and reports hit sparks and hit/guard sounds; the native game draws common sparks from common/fightfx (additive blend) and plays common sounds from common/common.snd. Checked in the real game: kfm's light kick on kfm shows its spark (capture). The web build does not draw sparks yet.
