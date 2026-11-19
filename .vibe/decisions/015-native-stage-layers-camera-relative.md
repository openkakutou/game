---
date: 2026-10-30
status: accepted
---
# Native stage layers: camera-relative parallax, web placement rules otherwise

**Context:** Backlog 019 adds the stage's background layers to the native match. The web build keeps a fixed camera (decision 004), so its parallax deltas have no visible effect; the native renderer already scrolls fighters against the midpoint of both fighters.
**Decision:** Native layers follow web rules (split behind/front at `layerNo` 1, stable on file order, top-left placement with the sprite axis ignored, a 32 px placeholder tile for an unloadable sprite, nothing for a blank or missing animation, no tiling or blend modes). Their position is `stage.ResolveParallaxPosition` against the negated native camera, so delta 1 shifts like a fighter and delta 0 stays fixed. A stage with no elements keeps the flat sky/floor backdrop.
**Reason:** Reusing the stage library's own parallax function keeps one formula; scrolling against the existing camera keeps layers and fighters coherent. With the camera at the origin the picture is identical to the web build.
**Rejected alternatives:** Fixed camera for layers (fighters would slide over a static backdrop); drawing the flat backdrop under layers always (hides transparency and differs from web).

**Amendment (2026-11-19, backlog 056):** Ignoring the sprite axis put every layer's left edge on the screen centre, so the Training stage (axis 364 on a 729 px backdrop) left the view's left half black. Native layers now follow Ikemen GO (commit `54d4148a`) instead of the web rules: the sprite axis sits on `start` (anim.go:1095), positions and sizes are scaled from the stage's `localcoord` to the view (stage.go:1772), a stage shorter than the view is bottom-aligned (camera.go:149), and `tile` repeats the sprite. The camera is no longer the fighters' midpoint: it starts at `startx` and moves only when a fighter comes within `tension` of a view edge, eases there and stops at `boundleft`/`boundright` widened by `zoomout` (camera.go:193, 314, 361-583, zoom off). Not covered until the stage library parses them: `startx` (0 used), `tension` (Ikemen's default 50 used), `scalestart`/`xscale`.
