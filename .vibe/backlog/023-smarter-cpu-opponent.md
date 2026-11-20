---
status: blocked
---
# Smarter CPU opponent with difficulty levels

## Description
The CPU is a minimal approach-and-attack bot. Add blocking, anti-air, varied attack choice and a difficulty setting chosen on the match setup screen.

## Acceptance Criteria
- [ ] Easy/Normal/Hard selectable in setup
- [ ] CPU blocks and reacts to attacks according to level
- [x] Behaviour is deterministic under a fixed random seed for tests

## Notes
Cross-repo: blocking needs engine#023 (guarding).
Combat-completeness milestone, roadmap decision `036`: comes after the engine steps.

## Blocked
2026-11-20 — Game side done (decision 030): the CPU jams buttons as Ikemen GO and gives the character's state machine its AI level (Difficulty 1-8 in Gameplay options, `-ailevel`, `-p<n>.ai`), replacing Easy/Normal/Hard in the setup. Left to the engine: characters' own AI files (kfm_zss AI.zss needs engine item 056), blocking (engine item 023). `TestAILevel_KungFuMansOwnAIPlaysOnlyWithALevel` is skipped until then. Not yet: Ikemen GO's arcade difficulty ramp; the web build's CPU always plays at level 4.
- 2026-11-21: engine v2.27.0 runs kfm_zss's own AI: at a level it acts on its own (test un-skipped). Left: blocking (engine item 023), screen edges from the match camera (item 074).
