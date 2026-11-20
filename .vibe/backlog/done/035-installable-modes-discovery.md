---
status: done
depends_on: [031]
---
# Discover installable modes from `modes/`

## Description
Roadmap decisions `030`, `031`: modes are discovered like content. Native scans `modes/`; the web build generates its manifest at build time from the same folder.

## Acceptance Criteria
- [ ] Dropping a valid mode file into `modes/` makes it available, native and web (after rebuild)
- [ ] An invalid mode file is reported with its error and left out, the others still load

## Done (2026-11-20)
Native scans `modes/` (ADR 022). Web build-time manifest: item 068.
