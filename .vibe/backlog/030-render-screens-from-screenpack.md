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
- Next: options/controls, pause, character select, stage select, match setup, VS, results screens; then menu sounds and screen music; then screen-level transitions.
- Open points from part 1: `blinkMs` read as the half-period (format.md does not say; settle in the screenpack docs); no game version string to show (`TitleState.Version`); WOFF2 fonts refused (x/image/opentype); shortcuts left: rect radius, tiled `scroll`, kerning, horizontal menus, text-element transitions, full-atlas glyph drop; single texts placed in a box of their own width (align only matters between menu entries — check against format.md); `ui` imports `game` for `ResolvePath` (watch for a cycle when the core needs `ui` for the web, item 067).
