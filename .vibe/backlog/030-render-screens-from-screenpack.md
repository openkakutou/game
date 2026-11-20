---
status: in_progress
---
# Render the game's screens from an OpenKakutou screenpack

## Description
Roadmap decision `029`: title/main menu, character and stage select (ordered by the screenpack's sort rule, roadmap decision `034`), match setup, VS, options, controls, pause, move list, results, continue and victory screens are rendered from the resolved screenpack (via the `screenpack` library: native Go import, WASM on web), identically in both builds, not with HTML components. The default screenpack ships with the game; the selected one comes from settings (item 029).

Blocked on the `screenpack` repo's first release (roadmap backlog `019`).

## Acceptance Criteria
- [ ] Every screen listed above renders from the default screenpack, web and native
- [ ] Switching screenpack in options changes the screens; a screenpack missing a screen falls back to the default one's
- [ ] An invalid screenpack is reported and the default is used instead
- [ ] Visual regression baselines cover each screen on the default screenpack

## Progress
- 2026-11-20 — part 1 (ADR 025): `native/internal/ui` layout (units, anchors, images, tiling, sprite frames, text styles via opentype, menus, element `in` transitions, blink) and gfx draw-list painter; title and main menu drawn from the screenpack (built-in default via `screenpack.DefaultFS`, or `screenpacks/default` in the game folder).
- 2026-11-20 — part 2: options (pages and controls bindings), pause, match setup, VS, results drawn from the pack; glyph atlas multi-page; menu first entry at the pivot (spec); refused pack folder reported. Default pack layout fixed in screenpack v0.6.1; format gaps logged as screenpack backlog 009. Results gained a cursor (Rematch / Back to selection).
- 2026-11-20 — part 3: character select grid (2-D navigation, sort rule from the pack, small/large portraits from sprites 9000,0/9000,1, display name from [Info] displayname → name → folder), stage select (name, arrows; preview is a placeholder panel), portraits on VS and results. Default select layout fixed in screenpack v0.6.2. Reviewed: no bug found.
- 2026-11-20 — part 4: menu sounds (move/confirm/cancel/error cues from the core), screen music (root/per screen, null, same-file continuity, handed to the stage music in matches), screen fade-in; menu sounds on their own channel group. Reviewed; 3 findings fixed (sounds cut at a match change, failing music retried every frame, silent pause close). Shortcuts: loopStartMs, out transitions, non-fade screen transitions.
- Next: error screen (format gap, screenpack 009); random cell and palette text (need flow support); stage preview; first-frame portrait decode spread over frames for big rosters.
- Open points from part 1: `blinkMs` read as the half-period (format.md does not say; settle in the screenpack docs); no game version string to show (`TitleState.Version`); WOFF2 fonts refused (x/image/opentype); shortcuts left: rect radius, tiled `scroll`, kerning, horizontal menus, text-element transitions, full-atlas glyph drop; single texts placed in a box of their own width (align only matters between menu entries — check against format.md); `ui` imports `game` for `ResolvePath` (watch for a cycle when the core needs `ui` for the web, item 067).
