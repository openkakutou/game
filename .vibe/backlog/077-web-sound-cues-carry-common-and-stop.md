---
status: todo
---
# Web sound cues carry `common` and `stop`

## Description
Found by the audit of 2026-11-21 (medium). The snapshot sends a `game.Cue` as `side`, `group`, `sample` only (`core/tick.go:89-91`), and the web plays every cue from the fighter's own `.snd` (`match-renderer.ts:579-582`). An F-prefixed PlaySnd (kfm.cns:220, 226, 1499: common sounds) plays the character's own sample with the same numbers, and a StopSnd cue (group 0, sample 0, `Stop`) plays sound 0,0 instead of stopping one. Item 055 did this for native only.

🔒 MUST: as in Ikemen GO (roadmap decision `038`).

## Acceptance Criteria
- [ ] The snapshot's cue carries `common`, `stop`, `channel` and `lowPriority`; the bridge validates them
- [ ] On the web, a common cue plays from the common sound file (or nothing when it is missing), a stop cue stops the fighter's channel, and neither plays the character's sample by mistake
- [ ] Lifebar cues are not sent as fighter sounds
