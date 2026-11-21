---
status: todo
---
# Character .act palettes

## Description
The desktop game draws every sprite with the palette embedded in the sprite file and never reads a character's `.act` palettes (`pal1` to `pal12` in the `.def`). Ikemen GO (char.go:4743-4800, commit `54d4148a`) replaces, for an SFF v1 character, the first physical palette (sprite 0's and every sprite sharing it) with the selected `.act` (pal 1 unless `Pal.Defaults` or the player picks another); sprites with their own palette keep it. SFF v2 characters select among their own palettes. Seen on the corpus Ryu (SVC): his hadouken shows in a colour that looks wrong.

MUST: Ikemen GO behaviour (roadmap decision `038`).

## Acceptance Criteria
- [ ] An SFF v1 character's shared-palette sprites use its selected `.act` palette, own-palette sprites keep theirs
- [ ] An SFF v2 character uses its selected palette
- [ ] The palette choice of the select screen (once it exists) picks which one
- [ ] Checked in the game against Ikemen GO (fidelity video) on a corpus SFF v1 character

## Notes
Needs the `sff` library to tell which v1 sprites share the first palette (its palette chain, `v1PaletteSource`, is unexported today).
