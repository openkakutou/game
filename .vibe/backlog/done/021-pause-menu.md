---
status: done
---
# In-match pause menu

## Description
There is no way to pause or leave a match. Add a pause menu (resume, restart match, options, move list, return to character select, quit) in both the web and native builds, reachable from keyboard and gamepad.

## Acceptance Criteria
- [x] Pause freezes the simulation and audio
- [x] Resume, restart, back to select and quit work on both targets (web has no quit: a browser tab has no program to exit)
- [ ] Options changed from the pause menu apply when the match resumes (audio, controls) or on the next match (gameplay) — moved to item 029, which has no options to show yet
- [x] Gamepad Start and keyboard Escape open it

## Notes
Entries and title follow Ikemen GO's default [Pause Menu] (resources/defaultMotif.ini at 54d4148a): Continue, Character Change, Exit, plus Restart; Start closes it as menu.cancel.key. Move list: item 042. Quick versus (native) has no Character Change.
