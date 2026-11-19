---
status: done
depends_on: [037]
---
# Lifebars from `lifebars/` with a selection setting

## Description
Roadmap decision `031`: each folder under `lifebars/` is one lifebar in MUGEN/Ikemen GO format (`fight.def` + its files), chosen independently of the screenpack and stored in `appearance.json`. `lifebars/default` ships with the game.

🔒 MUST: a lifebar taken from a MUGEN or Ikemen GO screenpack's `data/` works when dropped as is into `lifebars/<name>/`.

## Acceptance Criteria
- [x] Every valid lifebar folder is listed; a broken one is reported and skipped
- [x] The match HUD uses the selected lifebar, falling back to `default` when the selection is missing
