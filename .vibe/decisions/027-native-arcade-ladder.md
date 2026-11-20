---
date: 2026-11-20
status: accepted
---
# Native arcade: ladder, continue and victory

**Context:** Backlog item 044, native first: a built-in Arcade mode — one character, a ladder of CPU opponents with an optional final boss chosen through the roster filter (item 036), a continue screen and a victory screen. The mode format already has `opponents.kind: ladder` with `count`, and the outcomes `nextOpponent`, `continue`, `gameOver` (ADR 021 ran only `results`).

**Decision:**
- Format: `rules.opponents` gains an optional `final` (one character folder name) fought after the `count` ladder fights; validation like other names, an uninstalled `final` is reported at startup (as roster names) and skipped.
- Ladder: drawn when the mode is entered, after the player's pick: `count` opponents at random without repeats from the roster filter's opponent pool minus `final` and minus the player's own character (the player's character is allowed only when nothing else is left); fewer candidates than `count` shortens the ladder. Each fight's stage is a random visible stage. The random source is injected (seeded from the clock in the game, fixed in tests).
- Flow for Arcade: character select (p1) → versus → match; `onWin: nextOpponent` → versus → match with the next opponent, and after the last one the `victory` screen, then the main menu; `onLose: continue` → the `continue` screen: a 10-second countdown (one `tick` per second), Yes = refight the same opponent, No or timeout = `gameOver` sound then the main menu. `gameOver` as an outcome goes straight to the main menu.
- Screens from the pack: `continue` (portrait of the player's character, message "Continue?", countdown, Yes/No menu laid out horizontally — horizontal menus are added to `internal/ui` for this) and `victory` (winner portrait, name, message, `durationMs` then the main menu, confirm skips).
- Built-in `modes/arcade.jsonc`: p1 human vs p2 CPU, 1 character each, ladder `count` 6, no `final`, rounds 3, time 99, no roster filter. A content creator adds a boss by editing the file (`final` plus `characters.select.exclude`).

**Reason:** Uses the opponent pool from item 036 and the outcomes the format already lists; a random ladder without repeats is the MUGEN/Ikemen GO arcade behaviour players expect when no order file exists, and a single `final` covers the boss case without a ladder file format.

**Rejected alternatives:** A ladder order file per mode (no content for it yet, and the roster filter already selects); Ikemen GO's `select.def` arcade orders (out of scope: screenpack/select files are not read, roadmap decision 029); letting the player pick each stage (arcade picks for the player in MUGEN and Ikemen GO).
