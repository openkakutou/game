---
status: done
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
- [x] Each point is fixed or explicitly kept with its reason

## Done
Fixed:
- Dead code: `Pads.ID`, `mode.Load` (its test now checks that `Parse` names the file), `Announcer.finish` (read as `result.Outcome`), the `_ = i` in `currentFrame`. `TestCore_ImportsNoFileAccessNorDrawing` now also checks `mode`, `lifebar` and `bmfont`.
- `CommonSounds.Warnings()` became `Err()`, and `FindCueSound` returns it: the audio log now says why a common or lifebar sound file cannot be used (it was computed and never printed).
- `Setup.CPU` and `QuickVersus.CPU` are gone: a side is CPU-driven when it has an AI level (`Setup.CPU()`), each setup builder sets the level directly.
- The textured quad (`texturedQuad`, which also drops a closure built per tinted sprite) and the mirrored sprite offset (`spriteOffsetX`, tested) each live once; the renderer's screenpack screen is looked up from `Flow.PackScreen()` (guard test `TestPackScreens_EachScreenHasOneName`).
- `SavePNG` checks the `.png` suffix before creating the file and returns the close error (`writePNG`, tested).
- vite 8.3.8 and vitest 4.1.12: `npm audit` reports 0 vulnerabilities (npm 10.9's update crashes on a peer-set bug; npm 11 was used).
Already gone: `resolveCharacter` / `resolveStage` in `src/main.ts`.
Kept, with reason:
- The `"default"` names (`game.defaultName`, `lifebar.DefaultName`, `ui.DefaultPack`) name three different things that Ikemen GO happens to call alike; `game` cannot import `ui` (ui's tests import game).
- The native default bindings copied from `src/input/key-bindings.ts`: the web and native builds are two codebases (roadmap decision 022); item 069 moves web settings through the core, which removes the copy.
- The web options only tests pass (`roundOptions`, `timeLimitOptions`, `roundResultDelayMs`, `onAssignmentChange`) and the snapshot fields nothing reads: the web screens and drawing data move to the core flow (items 065, 067, 069), where they go away; rewriting their tests now would be wasted.
