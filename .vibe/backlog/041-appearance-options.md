---
status: done
depends_on: [029, 039, 040]
---
# Appearance options: choose screenpack and lifebar

## Description
Roadmap decisions `031`, `032`: an Appearance section of the options lets the player pick the screenpack and the lifebar among the installed ones, with a preview, saved in `config/appearance.json`.

## Acceptance Criteria
- [ ] Both choices persist across restarts, native and web
- [ ] The preview shows the candidate before it is applied
- [ ] A choice whose folder was removed falls back to `default` with a visible notice

## Done (2026-11-20)
Native (ADR 026): Appearance page, preview by drawing the page with the candidate pack and a sample lifebar. Web: item 069.
