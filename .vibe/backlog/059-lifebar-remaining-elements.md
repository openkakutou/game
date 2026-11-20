---
status: todo
depends_on: [039]
---
# Lifebar: remaining fight.def elements

## Description
Item 039 draws a lifebar's `[Lifebar]` and `[Powerbar]` bars (bg0-2, mid, front, top, range, layerno, facing, scale), places `[Name]` and `[Time]` in the built-in font, and steps the trailing fill like Ikemen GO. Still missing, each checked against Ikemen GO's `fightscreen.go` (commit 54d4148a):
- `[Face]`, `[Round]` (round and KO announcements, sounds), `[WinIcon]`, `[Combo]` (its counter in the lifebar fonts, from item 049), `[Action]`: the built-in round banner stands in, and win counts are not shown with a lifebar.
- Per-value element variants (`front.<n>`, `bg0.<n>`, `red`, `value`, `warn`, `shift`, `scalefill`, `palfx`), `vfacing`, `window`, `doubleres`.
- Team layouts (`[Simul Lifebar]`, `[Turns Lifebar]`, `[Tag …]`), guard and stun bars, the lifebar's own `.snd` and fightfx.
- Ikemen GO's `-fight <def>` command-line option (backlog 058).
- The move type the engine does not expose: a drop in life stands in for Ikemen GO's `getHit` (decision 020).
- Original art for the shipped `lifebars/default` (the test fixture is solid bars) and the web build.

🔒 MUST: a lifebar from a MUGEN or Ikemen GO screenpack's `data/` shows the same elements as in Ikemen GO.

Reading and resolving these elements is built in the shared `lifebar` library (roadmap decision `046`, `lifebar` items 003-005, then item 061 here); this item keeps the drawing.

## Acceptance Criteria
- [ ] A real Ikemen GO lifebar (e.g. V-HD Bars) shows faces, round and win icons, combo counts as in Ikemen GO
- [ ] Per-value variants switch at the same life thresholds as Ikemen GO
