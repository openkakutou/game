---
date: 2026-11-20
status: accepted
---
# Native screens drawn from the screenpack through a cgo-free layout

**Context:** Backlog item 030: every menu screen is drawn from the resolved OpenKakutou screenpack (roadmap decision 029), identically on web and native. Today the native game draws its screens with hard-coded positions and a bitmap font. The `screenpack` library (v0.6.0) loads, validates and resolves a pack and embeds the default one (`DefaultFS`).

**Decision:**
- A new cgo-free package `native/internal/ui` turns a resolved screen plus what the game shows (menu entries, cursor, names, values, elapsed time) into a flat draw list: filled rects, image quads (source rect, destination rect, opacity, tiling), and laid-out text runs (glyph positions per font and pixel size, color, outline, shadow). Units, anchors, pivots, offsets, sizes, transitions, blinking, scrolling and menus follow `screenpack` `docs/format.md`. Text is measured with `golang.org/x/image/font/opentype` (pure Go), so layout is unit-tested without a window, and the same package can later feed the web build through the game core (items 065/067).
- `internal/gfx` only executes the draw list: images become textures once, fonts become glyph atlases per (font, pixel size) once, and the draw list's backing arrays are reused frame to frame (no per-frame heap allocation).
- The pack used is `screenpacks/default` in the game folder when present and valid, else the built-in `screenpack.DefaultFS()`, with a notice on stderr when the folder is invalid. Choosing another pack is item 040/041.
- Screens move over one at a time: title and main menu first; the others keep today's drawing until their turn, in this item.
- Menu sounds and screen music from the pack are part of this item, after the drawing.

**Reason:** A pure-Go layout is testable, keeps gfx thin, and is the piece the web build will share; resolving per (font, size) atlases once keeps the frame budget.

**Rejected alternatives:** Drawing straight from the screenpack model inside gfx (untestable without a GL context, and nothing reusable for the web); SDL2_ttf (a cgo dependency for what pure Go already does, and not available to the web core).
