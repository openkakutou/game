---
status: done
---
# Screen edges from the match camera

## Description
Ikemen GO's camera is part of the simulation: each tick it sets the screen edges `sys.xmin` / `sys.xmax` (system.go:2871-2881), which keep the fighters on screen and which the edge triggers (`BackEdgeDist`, `FrontEdgeBodyDist`, …) read. Here the camera lives in the renderer and the web core only, so the match gives the engine no screen (`engine.TickConfig.Screen`, v2.27.0) and the edges fall back to the stage bounds: kfm_zss's AI reads its distance to the stage bounds instead of the screen edges.

🔒 MUST: as Ikemen GO (commit `54d4148a`).

## Acceptance Criteria
- [x] The match keeps its own camera (Ikemen GO's 320-unit game width) and passes the screen edges to the engine each tick, in the native and web builds
- [x] Fighters cannot walk off screen, as in Ikemen GO
- [x] The renderer draws from the same camera

## Notes
- 2026-11-21: done (decision 031). The web build reads the same camera through the core snapshot's camX.
