---
status: todo
---
# Screenpack rendering polish (native)

## Description
Left out of item 030 (ADR 025), each marked `shortcut:` in `native/internal/ui`:
- rect `radius` drawn square; tiled image `scroll` ignored; text-element transitions ignored (`element.go`)
- whole-screen transitions: only `fade`, never `out` (`layout.go`)
- no kerning (`text.go`)
- only vertical menus; options `tabs` not drawn (`menu.go`)
- character select `random` cell and palette text not drawn — need a random pick and a palette choice in the flow (`select.go`)
- stage preview is a plain panel (`select.go`)
- wrapped error message line spacing is OpenKakutou's (format has none) (`error.go`)
- menu music `loopStartMs` not applied (`internal/audio`)
- first select frame decodes up to a screenful of portraits at once (big rosters: spread over frames)

## Acceptance Criteria
- [ ] Each point above is drawn as `screenpack` `docs/format.md` describes, or recorded as a deliberate limit in an ADR
