---
status: todo
depends_on: [065]
---
# Web settings and options screen

## Description
Split from item 029 (2026-11-20), which shipped the native half (ADR 023). The web build stores the same concerns as native in browser storage, one key per concern (`gameplay`, `audio`, `video`, later `controls` and `appearance`), with the same fields, defaults and fallback rules, and shows the same options screen through the core flow (item 065).

## Acceptance Criteria
- [ ] Each concern persists and restores independently in the browser
- [ ] A missing or corrupt stored value falls back to defaults for that concern only, with a visible notice
- [ ] Options reachable from the main menu and the pause menu, keyboard and gamepad
