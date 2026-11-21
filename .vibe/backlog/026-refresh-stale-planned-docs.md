---
status: todo
---
# Refresh stale Planned entries in README and CLAUDE.md

## Description
README lists the native build as planned while the native desktop game exists, and CLAUDE.md has a Planned section for modules that shipped. Resync both.

## Acceptance Criteria
- [ ] README Planned lists only what is truly not built (Android, Windows/macOS verification)
- [ ] CLAUDE.md Planned section removed or corrected
- [ ] /vibe:sync run clean
- [ ] `CLAUDE.md` Architecture no longer says "Early scaffold stage" nor shows a 6-file tree (audit 2026-11-21, high: every session loads it)
- [ ] `CLAUDE.md` cites the roadmap's decisions 020/021/022 (Wails, Tauri, go-gl) as roadmap decisions, not as local `.vibe/decisions/` files
- [ ] `.vibe/index.md` names `native/internal/core` as the core and lists `internal/core` and `internal/mode`; `.vibe/glossary.md` no longer cites `src/hud/hud-view-model.ts`
- [ ] A decision records that the core plays player 2 and validates the snapshot, superseding the parts of decisions 004, 009 and 010 that describe `engine-bridge.ts`, `hud-view-model.ts` and `src/cpu/`
