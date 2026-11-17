---
status: todo
depends_on: [037]
---
# Android: pick the game folder

## Description
Roadmap decision `033`: on first launch the Android app asks the player to pick the game folder through the system folder picker (Storage Access Framework), keeps that access, and reads content through it.

Blocked on roadmap backlog `014` (Android go/no-go).

## Acceptance Criteria
- [ ] The picked folder is remembered across launches
- [ ] The same folder copied from a PC plays unchanged
- [ ] Losing access to the folder asks to pick it again
