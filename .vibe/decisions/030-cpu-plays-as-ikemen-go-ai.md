---
date: 2026-11-20
status: accepted
---
# The CPU plays as Ikemen GO's AI: button jamming plus the character's own AI

**Context:** Backlog item 023 asked for a smarter CPU with Easy/Normal/Hard in the match setup. The CPU was an OpenKakutou approach-and-attack bot, and the engine's `AILevel` trigger always read 0, so a character's own AI (its states gated on `AILevel`, like kfm_zss's AI.zss) never ran.
**Decision:** A CPU side gets an AI level, 1 to 8: the Gameplay option Difficulty (default 4, Ikemen GO's Options.Difficulty), overridden by `-ailevel` for the run or `-p<n>.ai <level>` in quick versus. The match hands it to the side's state machine (engine v2.26.0 `Context.AILevel`), and the CPU's own input is Ikemen GO's button jamming (`AiInput.Update`, input.go:1955-1997, commit 54d4148a): a random direction and random buttons held for random times, more often at a higher level. The draw is injectable, so a seeded draw replays the same inputs. Ikemen GO's `d` and `w` buttons are not jammed (OpenKakutou characters have none), and the arcade's difficulty ramp is not applied yet.
**Reason:** 🔒 MUST behave as Ikemen GO; MUGEN and Ikemen GO characters ship their AI in their own files and expect the engine to supply the level and the jamming. Easy/Normal/Hard in the setup would be OpenKakutou-only behaviour, and the eight levels already cover it.
**Rejected alternatives:** Improving the approach-and-attack bot with blocking and anti-air (a behaviour no character asked for, and it would override characters' own AI); three levels in the match setup (diverges from Ikemen GO's 1-8 difficulty and its options screen).
