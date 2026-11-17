---
status: todo
depends_on: [039]
---
# Bitmap font text in the HUD and lifebars

## Description
Lifebars draw names, timers and combo counters with MUGEN/Ikemen GO `.fnt` fonts. Render them through the `fnt` library (roadmap decision `035`).

Cross-repo: waits for the `fnt` repo's first release (roadmap backlog `023`).

🔒 MUST: `.fnt` v1 and v2 as used by real MUGEN 1.0/1.1 and Ikemen GO lifebars.

## Acceptance Criteria
- [ ] Names, timer and combo counter of a real lifebar render with its own fonts
- [ ] A missing font falls back to a default with a visible notice
