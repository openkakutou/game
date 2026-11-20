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
