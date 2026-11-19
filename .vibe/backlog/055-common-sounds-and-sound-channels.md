---
status: todo
---
# Common Sounds and Sound Channels

## Description
engine v2.18.0 sound events say where a sound comes from and how it uses channels (engine#040, engine `docs/wasm.md` "Sound events"). The game must follow them:
- `common` true reads `group,sample` from the common sound file in `common/` (roadmap decision `031`). Otherwise the sound comes from the character's own `.snd`.
- `channel` ≥ 0 stops the fighter's sound still playing on that channel. With `lowPriority`, the new sound is dropped instead while the channel is busy. `channel` -1 never interrupts. This follows Ikemen GO (sound.go:1243-1263, commit `54d4148a`).

🔒 MUST: as in Ikemen GO (roadmap decision `038`).

## Acceptance Criteria
- [ ] A common sound plays from `common/`. A missing common file is silence plus one warning, not a crash
- [ ] Channel interruption and `lowPriority` behave as above, in the native and web builds
- [ ] Checked in the real game with the fidelity video

## Notes
Shipping a common sound file is still game#038 (licence). Found by engine#040 (2026-11-19).
