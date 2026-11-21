---
status: todo
---
# Render projectiles and hit reactions

## Description
Once engine reports projectiles, hit pause and guard states, draw them in both builds: projectile sprites and animations, hit sparks, guard feedback, and screen shake.

## Acceptance Criteria
- [ ] Projectiles are drawn from the engine's response with correct facing
- [ ] Hit pause visibly freezes both fighters
- [ ] Missing projectile sprites fall back to a clear placeholder, not a crash

## Notes
Cross-repo: waits for engine#022 and #024.
Cross-repo: also waits for engine#029 (Explod: hit sparks). Combat-completeness milestone, roadmap decision `036`.

## Progress
2026-11-21 — Desktop: projectiles from engine v2.37.0 are drawn with their owner's animations and sprites, facing included (checked in the real game with a test-only kfm fireball), hit sparks shipped with item 038; hit pause already freezes both fighters' animations (the engine holds AnimElapsed). Left: the web build, a placeholder for a projectile whose sprite is missing (now skipped, no crash), screen shake.
2026-11-21 — Desktop draws helpers too (engine v2.41.0); a corpus Ryu (SVC)'s keyboard hadouken flies and hits kfm in the game. Seen: the fireball shows in an unexpected colour, likely the character's .act palettes not applied (to check).
