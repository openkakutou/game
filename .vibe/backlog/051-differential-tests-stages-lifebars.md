---
status: todo
---
# Differential tests against Ikemen GO for stages and lifebars

## Description
Roadmap decision `038`: Ikemen GO is the verification reference, end-to-end. For stages and lifebars that work in Ikemen GO, run the same content and the same scripted match in Ikemen GO and in `game`, and compare per tick: stage camera position and each background layer's position, visibility and animation element (BGCtrl included); lifebar values and element states (life, power, timer, combo counter, round indicators). Prefer comparing these values; compare screenshots only where no value can be read. Shares the trace-recording approach of engine#036.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO compatibility, verified through Ikemen GO; OpenKakutou-only content (3D characters, screenpacks) is out of scope.

## Acceptance Criteria
- [ ] Reference traces recorded from Ikemen GO for at least one real stage with parallax and BGCtrl, and one real lifebar
- [ ] `game`'s run compared per tick, first divergence reported with both values
- [ ] Runs in CI on the recorded traces

## Notes
Cross-repo: reuses the Ikemen GO trace recording from engine#036.
