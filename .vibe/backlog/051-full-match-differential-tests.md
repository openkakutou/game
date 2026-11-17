---
status: todo
---
# Full-match differential tests against Ikemen GO: states and screenshots

## Description
Roadmap decision `038`: Ikemen GO is the verification reference, end to end. Replay a whole recorded match (both players' inputs, frame by frame, same random seed, engine#036's replay) in Ikemen GO and in `game`, stepping both one frame at a time (no wall clock), and compare two things on the same replay (Product Owner choice, 2026-11-17):
- **States, every frame**: stage camera and each background layer's position, visibility and animation element (BGCtrl included); lifebar values and element states (life, power, timer, combo counter, round indicators). Engine state is compared by engine#036.
- **Screenshots, at regular intervals and on key events** (hit, round end), at the same internal resolution and scaling with filtering off, compared with a per-pixel tolerance; a failing comparison saves both images and their difference.

**No CPU**: both sides are driven by simulated inputs only (a scripted or recorded input sequence per player); the CPU opponent is never used, since its decisions are not the reference engine's and would make the two runs diverge for reasons unrelated to compatibility (Product Owner, 2026-11-17).

Only content that works in Ikemen GO (characters, stages, lifebars); OpenKakutou-only content (3D characters, screenpacks) is out of scope.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO compatibility, verified through Ikemen GO.

## Acceptance Criteria
- [ ] One recorded match on real acceptance content (roadmap decision `036`) replays identically in both engines
- [ ] State comparison reports the first divergent frame with both values
- [ ] Screenshot comparison reports differing frames with the two images and their difference
- [ ] Runs in CI on recorded traces and reference screenshots

## Notes
Cross-repo: needs engine#036 (input replay, random seed, Ikemen GO trace recording).
