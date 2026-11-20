---
status: done
depends_on: [037]
---
# Screenpacks from `screenpacks/` with a selection setting

## Description
Roadmap decisions `029`, `031`: each folder under `screenpacks/` is one OpenKakutou screenpack, the selected one stored in `appearance.json`, `screenpacks/default` shipped on disk and used as the per-screen fallback.

## Acceptance Criteria
- [ ] Every valid screenpack is listed; an invalid one is reported and skipped
- [ ] Changing the selection changes every screen's appearance; a screen missing from it falls back to `default`

## Done (2026-11-20)
Native (ADR 026). Web: item 069 (settings) and the core flow (065).
