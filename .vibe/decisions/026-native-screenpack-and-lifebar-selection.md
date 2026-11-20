---
date: 2026-11-20
status: accepted
---
# Native screenpack and lifebar selection

**Context:** Items 040 (screenpacks from `screenpacks/` with a selection setting) and 041 (Appearance options: choose screenpack and lifebar, with a preview), native first. Today the game draws from `screenpacks/default` or the built-in pack (ADR 025) and reads the lifebar name from `config/appearance.json` through `internal/lifebar` (decision 020).

**Decision:**
- `appearance.json` becomes the fifth settings concern of ADR 023 (`{"screenpack": "<folder>", "lifebar": "<folder>"}`, shared by every platform), with the same load, fallback, notice, preserve-unknown-keys, first-launch write and save-on-leave rules; the lifebar code reads the name from it instead of parsing the file itself.
- Screenpacks: every folder under `screenpacks/` (found case-insensitively) that loads and resolves is listed by folder name ignoring case; an invalid one is left out with its problems on stderr. The built-in pack is listed as `default` when `screenpacks/default` is absent or invalid. Each pack resolves against the default for missing screens and parts (ADR 025).
- A selection naming a folder that no longer exists (or is invalid) falls back to `default` with a notice on stderr and on the options screen; the file keeps the old name until the player changes the setting.
- Options gain an Appearance page (Gameplay, Audio, Video, Controls, Appearance, Back) with two rows changed with left/right: Screenpack and Lifebar. Preview: while the Screenpack row shows a candidate, the Appearance page itself is drawn with that candidate pack; while the Lifebar row shows a candidate, a sample lifebar (both sides full) is drawn under the menu with it. The choice applies, and is saved, when leaving the page; leaving with cancel also keeps it (as every options page).
- Switching packs re-sorts the character and stage lists with the new pack's sort rules; picks are reset when a mode is entered, so nothing else is remapped. Textures, glyph atlases, portraits and menu sounds of the old pack are released.

**Reason:** Reuses the settings machinery and the resolved-pack drawing already in place; drawing the page itself with the candidate is the cheapest faithful preview of a whole screenpack.

**Rejected alternatives:** A separate preview window or thumbnail per pack (screenpacks ship no thumbnail; the format has none); applying on every left/right press (switching packs and sorting on each press, and a half-chosen state saved on quit).
