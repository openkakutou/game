---
status: done
depends_on: [039]
---
# Bitmap font text in the HUD and lifebars

## Description
Lifebars draw names, timers and combo counters with MUGEN/Ikemen GO `.fnt` fonts. Render them through the `fnt` library (roadmap decision `035`).

Cross-repo: waits for the `fnt` repo's first release (roadmap backlog `023`).

🔒 MUST: `.fnt` v1 and v2 as used by real MUGEN 1.0/1.1 and Ikemen GO lifebars.

## Acceptance Criteria
- [x] Names, timer and combo counter of a real lifebar render with its own fonts
- [x] A missing font falls back to a default with a visible notice

## Notes
- 2026-11-20: names and timer render with the lifebar's fonts (v1 and v2 bitmap, palette and sprite banks), checked in the real game with Z-S Green (v1) and VHD (v2). The combo counter has no lifebar element drawn yet: it comes with `[Combo]` in item 059 and uses these fonts. TrueType `.def` fonts (e.g. `msgothic-tt36.def`) fall back to the built-in font with a notice.
