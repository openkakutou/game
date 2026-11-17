---
status: todo
---
# Options screen and settings split by concern

## Description
Roadmap decision `029`: settings live outside any screenpack, in one file per concern — `controls.json` (written by item 022), `gameplay.json` (difficulty, round time, life, rounds…), `audio.json` (volumes), `video.json` (resolution, fullscreen, vsync) — plus the selected screenpack. Native: `os.UserConfigDir()/openkakutou/`; web: browser storage, one key per concern. Add an options screen editing them; match setup defaults come from `gameplay.json`.

## Acceptance Criteria
- [ ] Each concern persists and restores independently on web and native
- [ ] A missing or corrupt settings file falls back to defaults for that concern only, with a visible notice
- [ ] Options screen reachable from keyboard and gamepad
