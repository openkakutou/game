---
status: todo
---
# Core: game logic without files or drawing

## Description
Step 1 of item 060 (roadmap decision `045`). Move the platform-independent game logic of `native/internal/game` (flow, match, CPU, camera, pause, roster rules) behind a core package that never touches the file system or the screen: content (character, stage, lifebar files) reaches it through a small interface the native build implements with the file system and the web build will implement with in-memory bytes. Drawing, input devices and audio output stay outside.

🔒 MUST: no behavior change — every native test passes unchanged; the native build plays the same.

## Acceptance Criteria
- [ ] The core package imports no `os`, `path/filepath` file access, SDL or GL
- [ ] The native build loads content through a file-system implementation of the content interface
- [ ] All native tests pass; a test runs a scripted match through the core with in-memory content
