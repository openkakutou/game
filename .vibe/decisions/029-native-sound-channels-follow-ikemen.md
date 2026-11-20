---
date: 2026-11-20
status: accepted
---
# Native sound effects: Ikemen GO's channel rules and 32-channel pool

**Context:** Backlog item 055: engine sound events carry PlaySnd's `channel` and `lowpriority`; decision 016 capped effects at 16 mixer channels and dropped a new effect when all were busy.
**Decision:** The pool grows to 32 channels (Ikemen GO's default `WavChannels`), and the channel choice is `game.RequestSlot`, Ikemen GO's `SoundChannels.Request` (src/sound.go:1234-1330, commit 54d4148a) without per-sound priority: a fighter's sound on channel n cuts its previous sound on n unless `lowpriority`; channel -1 or ≥ 32 takes a free channel; when all are busy the oldest unnumbered sound is cut, then the oldest numbered one (`lowpriority` is dropped instead). Amends decision 016's channel count and its drop-when-full rule.
**Reason:** 🔒 MUST match Ikemen GO; keeping the rule in cgo-free `game` keeps it unit-tested.
**Rejected alternatives:** SDL_mixer channel groups per PlaySnd channel (the mapping is per fighter and per channel number, which groups cannot express); keeping 16 channels (diverges from Ikemen GO's channel-number bound).
