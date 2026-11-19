---
status: todo
---
# Use Stage Camera Tension and BG Scale

## Description
stage v0.17.0 exposes `[Camera]` `tension`/`startx`/`starty`, each BG element's `scalestart`/`scaledelta`/`xscale`, and the resolved `zoffsetlink` (stage#021). The native game's camera and layers (item 056) use Ikemen GO's defaults in their place. Use the real values, as Ikemen GO does (camera.go, stage.go, commit `54d4148a`). A stage without a `[Camera]` section reads `tension` 0: treat it as Ikemen GO's 50.

🔒 MUST: stages render as in Ikemen GO (roadmap decision `038`).

## Acceptance Criteria
- [x] The camera uses the stage's tension and start position
- [x] BG layers are drawn at their scalestart, with scaledelta and xscale applied where Ikemen GO applies them
- [x] The fighters' ground line uses the resolved zoffset
- [ ] Checked in the real game with the fidelity video

## Notes
Follows item 056 and stage#021 (2026-11-19).

## Progress
2026-11-19: tension, startx, scalestart and the resolved zoffset are used (stage v0.17.0). scaledelta (zoom only) and xscale (parallax only) are not applied: the game has neither yet. Left: the fidelity-video check in the real game, which needs the SDL2 build tools (pkg-config) and ffmpeg, absent on this machine.
