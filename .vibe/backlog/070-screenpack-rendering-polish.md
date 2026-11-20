---
status: todo
---
# Screenpack rendering polish (native)

## Description
Left out of item 030 (ADR 025), each marked `shortcut:` in `native/internal/ui`:
- text-element transitions ignored (`element.go`)
- whole-screen transitions: only `fade`, never `out` (`layout.go`)
- no kerning (`text.go`)
- only vertical menus; options `tabs` not drawn (`menu.go`)
- character select palette text not drawn — needs a palette choice in the flow (`select.go`)
- stage preview is a plain panel (`select.go`)
- wrapped error message line spacing is OpenKakutou's (format has none) (`error.go`)

## Acceptance Criteria
- [ ] Each point above is drawn as `screenpack` `docs/format.md` describes, or recorded as a deliberate limit in an ADR

## Notes
- 2026-11-20: done — rect `radius` (rounded corners, capped at half the short side), tiled image `scroll`, menu music `loopStartMs` (the track plays once, then restarts from the loop point).
- 2026-11-20: the Appearance page's sample lifebar is drawn in the page's lower half, no longer over the heading.
- 2026-11-20: portraits are decoded at most 8 per frame, a big roster's select screen fills in over a few frames.
- 2026-11-20: the grid's `random` cell follows the last character; confirming on it picks a visible character at random, with the `random` sound.
