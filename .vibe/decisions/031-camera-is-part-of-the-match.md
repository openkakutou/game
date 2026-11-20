---
date: 2026-11-21
status: accepted
---
# The camera is part of the match simulation

**Context:** Backlog item 074. Ikemen GO's camera is simulated: each tick it sets the screen edges (`sys.xmin`, `sys.xmax`, system.go:2871-2881) that keep the fighters on screen (`xScreenBound`) and that the edge triggers read. Here the native renderer and the web core each followed the fighters with their own camera in view pixels, so the simulation had no screen at all.
**Decision:** `Match` owns a `StageCamera` in stage units (scale 1, the stage's localcoord width), snapped at each round start and moved after each tick, and gives the engine the screen edges each tick (engine v2.28.0 `TickConfig.Screen`): the camera's view less the stage's `[Bound]` screenleft/screenright (stage v0.18.0), 15 at a 320-wide localcoord by default. The renderer and the web core draw from `Match.CameraX()` scaled to view pixels.
**Reason:** 🔒 MUST behave as Ikemen GO: fighters cannot leave the screen, and characters' AI reads distances to its edges.
**Rejected alternatives:** Letting each view keep its camera and passing its edges to the match (the simulation would depend on the window size and differ between native and web).
