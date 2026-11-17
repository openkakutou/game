---
status: todo
---
# Options screen and settings split by concern

## Description
Roadmap decision `029`: settings live outside any screenpack, in one file per concern — `controls.json` (written by item 022), `gameplay.json` (difficulty, round time, life, rounds…), `audio.json` (volumes), `video.json` (resolution, fullscreen, vsync) and `appearance.json` (selected screenpack and lifebar). Native: the `config/` folder inside the game folder, next to the executable (roadmap decision `032`, which replaces `029`'s `os.UserConfigDir()`); web: browser storage, one key per concern. `video.json` and `controls.json` hold one section per platform (`windows`, `macos`, `linux`, `android`, `web`), the other files are shared (roadmap decision `033`); `config/` is created on first launch. Add options screens editing gameplay, audio and video (controls: item 022, appearance: item 041); match setup defaults come from `gameplay.json`.

## Acceptance Criteria
- [ ] Each concern persists and restores independently on web and native
- [ ] A missing or corrupt settings file falls back to defaults for that concern only, with a visible notice
- [ ] Options screen reachable from keyboard and gamepad
