---
date: 2026-11-20
status: accepted
---
# Native rebindable controls

**Context:** Backlog item 022, native first: each player rebinds keyboard keys and gamepad buttons from a controls screen, saved in `config/controls.json` with one section per platform (roadmap decisions 032, 033), conflicts shown and resolved explicitly.

**Decision:**
- Actions per player: keyboard `up`, `down`, `left`, `right`, `a`, `b`, `c`, `x`, `y`, `z`; gamepad `a`, `b`, `c`, `x`, `y`, `z`, `start` (directions stay on the d-pad and left stick). Defaults are today's bindings (P1 WASD + F/G/H/R/T/Y, P2 arrows + J/K/L/I/O/P; pad A/B/X/Y/LB/RB/Start).
- File: `config/controls.json` = `{"<platform>": {"p1": {"keyboard": {...}, "gamepad": {...}}, "p2": {...}}}`; keys are stored by SDL scancode name (`"W"`, `"Up"`, `"Return"`), buttons by SDL game controller button name (`"a"`, `"leftshoulder"`, `"start"`). Loaded, validated and saved like the other concerns of ADR 023 (per-concern fallback with notice, unknown names = corrupt, other platforms and unknown keys preserved, written on first launch, saved when leaving the screen or quitting). The core keeps names only; the SDL layer turns them into scancodes and buttons once per change, never per frame.
- Screen: Options > Controls > Player 1 / Player 2 / Reset to defaults / Back; a player screen lists every action with its key and button; confirm on a row waits for the next key (keyboard column) or button (gamepad column, switched with left/right); Escape cancels the wait.
- Conflict: a key or button already bound to another action of either player (same device kind) is never assigned silently: the screen names it ("F is Player 2 A") and asks to confirm a swap (confirm swaps the two bindings) or cancel (nothing changes). Rows sharing a binding — possible only from a hand-edited file — are shown in red.
- Menus keep their fixed keys (arrows, WASD, Enter, Space, Escape, Backspace, and the pad's d-pad, A, B, Start), so a bad binding can never lock a player out of the menus.
- The web half joins item 069.

**Reason:** Names are readable in the file and stable across SDL versions; swap-or-cancel is an explicit resolution that never leaves two actions on one key; fixed menu keys keep the controls screen itself always reachable.

**Rejected alternatives:** Storing raw scancode numbers (unreadable, fragile); silently unbinding the other action (a hidden change the player did not ask for); menus following the rebinding (a player could lock themselves out).
