---
status: done
depends_on: [033]
---
# Title screen and main menu listing installed modes

## Description
Roadmap decision `030`: the game opens on a title screen, then a main menu listing every installed mode plus Options and Quit.

## Acceptance Criteria
- [ ] A new mode file appears in the main menu with no code change
- [ ] Options opens the options screens (items 022, 029, 041)
- [ ] Quit exits on native; on web it returns to the title

## Done (2026-11-20)
Native: title, main menu (installed modes then Quit), return to the menu from a mode (ADR 022). Options entry: added with the first options screen, item 029. Web: through the core flow, item 065.
