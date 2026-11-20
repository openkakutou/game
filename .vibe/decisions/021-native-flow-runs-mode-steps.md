---
date: 2026-11-20
status: accepted
---
# The native screen flow runs a mode's steps

**Context:** Backlog item 032 turns the hard-wired versus flow (select → stage → setup → match → result) into built-in screen types that run from parameters (roadmap decision 030), so modes (033, 044–046) are data, not code.

**Decision:** `game.Flow` is built from a `mode.Mode` and walks its `flow` steps: confirming a screen goes to the next step, cancelling goes back to the previous interactive step (never back into `versus` or `match`), cancelling the first step quits. Each screen reads its parameters from its step and the mode's rules: `characterSelect` lets only its `sides` pick; `matchSetup` offers rounds and time limit preselected to the rules' values, and the player 2 control row only when that side's control is `humanOrCpu`; a side whose control is `cpu` is always CPU-driven; without a `matchSetup` step the rules apply as written. A new `versus` screen shows both picked characters and moves on after 2 seconds or on confirm. After a match, `onWin`/`onLose` (from player 1's side) choose what follows; only `results` is supported until the ladder modes land, any other outcome is the error screen naming it. The built-in Versus mode is the embedded `versus.jsonc`. Native first: the web build gets the same flow through the game core (item 065).

**Reason:** One state machine walking steps keeps every screen reusable across modes, and keeps the player-facing Versus flow unchanged apart from the new VS screen the mode lists. Rules as setup defaults keep today's choice of rounds and time limit, which the format has no field to "leave open".

**Rejected alternatives:** One screen object per type behind an interface (more types for six screens, and gfx would still switch on the type to draw); treating fixed `rounds`/`timeLimitSeconds` rules as hiding those setup rows (would remove the rounds/time choice Versus offers today, a behavior change 033 forbids); implementing `continue`/`victory` now (they need the ladder rules of item 044, nothing exercises them yet).
