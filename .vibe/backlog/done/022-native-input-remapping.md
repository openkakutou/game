---
status: done
---
# Rebindable controls

## Description
Default controls are fixed. Let each player rebind keyboard and gamepad buttons from a controls screen, persisted between sessions, with conflict detection. Bindings are saved in `config/controls.json`, one section per platform (roadmap decisions `032`, `033`).

## Acceptance Criteria
- [ ] Controls screen for both players on web and native
- [ ] Bindings persisted and restored
- [ ] Conflicts are shown and resolved explicitly

## Done (2026-11-20)
Native (ADR 024): controls screen, config/controls.json, swap-or-cancel conflicts. Gamepad conflicts are checked within one player (each player has their own pad); keyboard conflicts across both players. The bound pad start is the in-match pause. Web: item 069. Not exercised with a real gamepad (Xvfb, no pad).
