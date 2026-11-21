---
status: todo
---
# Audit cleanups (low)

## Description
Low findings of the audit of 2026-11-21, worth taking when the code is next touched:
- Dead code: `Pads.ID` (`pads.go:88-94`), `mode.Load` (`mode/mode.go:193-200`, the only `os.ReadFile` in a package the core imports; extend `TestCore_ImportsNoFileAccessNorDrawing` to `mode`, `lifebar`, `bmfont`), `CommonSounds.Warnings()` read only by tests (the host should print it), the `_ = i` in `currentFrame` (`gfx/screens.go:631`), `Announcer.finish` (always `result.Outcome`).
- Fields that say the same thing: `Setup.CPU` and `Setup.AILevel` (`match.go:127`), set by hand in four places.
- Copies: the textured-quad code in `gfx.go` (three times), the mirrored-sprite placement in `screens.go` (four times), `resolveCharacter` / `resolveStage` in `src/main.ts` (already diverged), the `"default"` appearance name (three declarations), the native default bindings hand-copied from `src/input/key-bindings.ts`, the Flow-screen to pack-screen mapping (`menucues.go` and `gfx/packscreens.go`).
- `SavePNG` (`gfx.go:439`) truncates the file before checking the `.png` suffix and ignores the close error.
- Options only tests pass (`roundOptions`, `timeLimitOptions`, `maxHealth`, `roundResultDelayMs`, `onAssignmentChange`), and snapshot fields nothing reads (`timer`, `roundTimerTicks`, `roundsToWin`, `stateNo`) that the bridge still requires.
- `npm update vite vitest` for the dev-server advisories.

## Acceptance Criteria
- [ ] Each point is fixed or explicitly kept with its reason
