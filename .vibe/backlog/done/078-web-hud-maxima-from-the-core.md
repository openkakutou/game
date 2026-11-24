---
status: done
---
# Web HUD maxima come from the core

## Description
Found by the audit of 2026-11-21 (medium). `MAX_HEALTH = 1000` and `MAX_POWER = 3000` (`src/hud/hud-renderer.ts:13-16`) are hand copies of `DefaultHealth` / `MaxPower` (`native/internal/game/match.go:29-30`), and nothing ties them. Native fills the bar from each fighter's own `LifeMax` (`data.life`); the web divides by 1000, so a character whose `data.life` is not 1000 gets a bar that never reads full or is clamped at 100% too early.

## Acceptance Criteria
- [x] The core gives each fighter's life and power maxima (start result or snapshot), and the web HUD uses them for the fill and `aria-valuemax`
- [x] The two TypeScript constants are gone
- [x] A test with a character whose `data.life` is not 1000 pins a full bar at round start
- [x] The native power bar fills against the fighter's own `PowerMax` (engine v2.77.3), not `game.MaxPower` (`gfx/lifebar.go:166, 351`, `gfx/screens.go:798`): a 5000-power character's bar no longer reads full at 3000

## Done
2026-11-24. `Match.PowerMax` is each fighter's engine `Context.PowerMax()`; `game.MaxPower` is gone. The desktop bars (lifebar and built-in) fill with `powerLevel` against it; `-maxpowermode` and `-p<n>.power` follow Ikemen GO (system.go:4426-4432), so `-p<n>.power` above 3000 is accepted. The core snapshot carries each fighter's `lifeMax` and `powerMax`, the bridge refuses a non-positive one, and the web HUD fills and sets `aria-valuemax` from them; `MAX_HEALTH`, `MAX_POWER` and the HUD's `maxHealth` / `maxPower` options are gone. Tests: `TestMatch_EachFighterHasItsOwnPowerMax`, `TestMatch_StartPowerIsCappedAtTheFightersPowerMax`, `TestQuickVersusFrom_StartPowerAboveTheDefaultCapIsAccepted`, `TestPowerLevel_FillsAgainstTheFightersPowerMax`, `TestSnapshot_CarriesEachFightersLifeAndPowerMaxima`, the HUD's 1200-life / 5000-power test and the bridge's maxima test (red before the change).
