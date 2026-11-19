# Data models

## RosterManifestEntry
| Field | Type | Notes |
|---|---|---|
| id | string | Stable identifier for the character |
| files.def / files.air / files.sff / files.cns / files.cmd | string | Fetchable paths to the character's files, including its `.cmd` input command file |
| files.snd | string (optional) | Fetchable path to the character's `.snd` sound file (backlog item 013) — legitimately absent for a character with no sound effects |
| portrait | string | Fetchable path to a static preview image |
Defined in: `src/roster/manifest.ts`

## DiscoveredCharacter
| Field | Type | Notes |
|---|---|---|
| id | string | Same as the source `RosterManifestEntry.id` |
| portrait | string | Same as the source `RosterManifestEntry.portrait` |
| status | `"ok" \| "error"` | Discriminant |
| name | string | Present only when `status: "ok"` — the character's loaded name |
| message | string | Present only when `status: "error"` — why the character failed to load |
Defined in: `src/roster/discovery.ts`

## CharacterSummary / CharacterResult
| Field | Type | Notes |
|---|---|---|
| name | string | The loaded character's name |
| animations | `Animation[]` | Every `.air` action: its number, frames (sprite ref, position, time, flip, blend, Clsn boxes), and loop-start index |
| sprites | `SpriteGroup[]` | Sprite metadata (dimensions, axis/pivot) per `(group, image)` — no pixel data; see `resolveSprites` |
| stateDefs | `StateDefBlob[]` | Opaque combat state defs — never interpreted client-side (the game core runs the state machine) |
| sounds | `SoundGroup[]` | The character's decoded sound effects (backlog item 013) — empty for a character with no sound file, never an error |
| ok / character / error | `true` + `CharacterSummary`, or `false` + string | `CharacterResult` discriminated union — mirrors the WASM module's own `{character, error}` contract, never throws |
Defined in: `src/wasm/types.ts`

## Sound / SoundGroup (character sound effects)
| Field | Type | Notes |
|---|---|---|
| group, sample | number | Keys a `.cns` `PlaySnd` controller (and the game core's sound cues) references this sound by |
| sampleRate, channels, bitsPerSample | number | The decoded sample's own audio format |
| pcm | `number[]` | Interleaved by `channels`, normalized to signed 16-bit — a plain JSON number array, not a `Uint8Array` (unlike every other binary payload this app's bridges expose) |
| index | number | `SoundGroup` only — the group index shared by every `Sound` in `sounds` |
Defined in: `src/wasm/types.ts`; converted to Web Audio `Float32Array` channels by `src/audio/pcm-conversion.ts`, resolved from a triggered `SoundEvent` by `src/audio/sound-lookup.ts`.

## SpritePixelResult
| Field | Type | Notes |
|---|---|---|
| ok / pixels / width / height / error | `true` + decoded RGBA pixels + dimensions, or `false` + string | One `resolveSprites` request's result — flat, row-major, straight (non-premultiplied) alpha |
Defined in: `src/wasm/types.ts` (character), `src/wasm/stage-types.ts` (stage, as `StageSpritePixelResult`)

## StageManifestEntry
| Field | Type | Notes |
|---|---|---|
| id | string | Stable identifier for the stage |
| files.def | string | Fetchable path to the stage's `.def` file |
| portrait | string | Fetchable path to a static preview image |
Defined in: `src/stage/manifest.ts`

## DiscoveredStage
| Field | Type | Notes |
|---|---|---|
| id | string | Same as the source `StageManifestEntry.id` |
| portrait | string | Same as the source `StageManifestEntry.portrait` |
| status | `"ok" \| "error"` | Discriminant |
| name | string | Present only when `status: "ok"` — the stage's loaded name |
| message | string | Present only when `status: "error"` — why the stage failed to load |
Defined in: `src/stage/discovery.ts`

## MatchSetupConfig
| Field | Type | Notes |
|---|---|---|
| rounds | number | The chosen round count, a positive odd integer |
| timeLimit | `TimeLimitOption` | The chosen per-round time limit |
| player2Control | `"human" \| "cpu"` | Who drives player 2 (backlog item 007) — pre-selected to `"human"`, unlike the two fields above |
Defined in: `src/setup/setup-screen.ts`

## TimeLimitOption
| Field | Type | Notes |
|---|---|---|
| seconds | number | A fixed duration in seconds (positive integer) |
| (or) | `"unlimited"` | The literal tag — no timer at all, never a numeric sentinel |
Defined in: `src/setup/setup-screen.ts`

## StageSummary / StageResult
| Field | Type | Notes |
|---|---|---|
| name | string | The loaded stage's name |
| bgDef | `BGdef` | Stage-level settings: sprite sheet path, local coordinate space, ground level (`zOffset`), zoom range |
| elements | `BGElement[]` | BG layers: type (normal/parallax/anim), sprite/action reference, layer number, position, parallax delta, tiling |
| animations | `Record<string, BGAnimation>` | Every `[Begin Action N]` block, keyed by action number as a string, resolved via `resolveAnimationFrames` |
| stageBoundaries | `StageBoundaries` | The x-range characters may move within on this stage |
| musicFile | string | The stage's background music file path (backlog item 013) — empty when the `.def` declares no `[Music]` section |
| ok / stage / error | `true` + `StageSummary`, or `false` + string | `StageResult` discriminated union — mirrors the WASM module's own `{stage, error}` contract, never throws |
Defined in: `src/wasm/stage-types.ts`

## StartMatchRequest / MatchSnapshot (game core bridge)
| Field | Type | Notes |
|---|---|---|
| p1, p2, stage | string | `StartMatchRequest`: names of the `.def` files mounted into the core (`mountFiles`) |
| setup, p2cpu, viewWidth | `{rounds, timeLimitSeconds}` / boolean / number | Rounds (odd), time limit in seconds (0 = none), whether the core plays player 2, and the width the camera frames (the canvas width) |
| id, roundTimerTicks, roundsToWin | number | `StartMatchData` only: the match id, and the once-per-match HUD values |
| tick, round, timer | number | Ticks played this round, the round number, ticks left on the round timer |
| roundEnded, outcome | boolean / `"none" \| "ko" \| "doubleKo" \| "timeout" \| "draw"` | The round is over (no tick is accepted until `nextRound`) and how it ended |
| over, winner, roundWinner | boolean / `-1 \| 0 \| 1` | The match is decided and by whom; who won the round that ended (-1 for a draw or while fighting) |
| wins | `[number, number]` | Rounds won by each side |
| camX | number | The camera's position (middle of the view), in stage pixels |
| fighters | `[FighterSnapshot, FighterSnapshot]` | `{x, y, facing (1 right / -1 left), anim, animTime, stateNo, health, power}`; y is up, 0 the ground |
| cues | `SoundCue[]` | `{side, group, sample}` sounds triggered this tick, drained once |
Defined in: `src/wasm/core-types.ts`; produced by the Go side (`native/internal/core`) and validated, whole or not at all, by `src/wasm/core-bridge.ts`.

## RoundEndView / MatchEndView (result screen)
| Field | Type | Notes |
|---|---|---|
| round | number | `RoundEndView` only — the round number that just ended |
| winner | `"p1" \| "p2" \| "draw"` | The snapshot's `roundWinner`/`winner`; no winner (-1) reads as `"draw"` |
Defined in: `src/result/outcome.ts`; built by `deriveRoundOutcome`/`deriveMatchOutcome`, rendered by `src/result/result-screen.ts`'s `createResultOverlay`.

## RawPlayerInput / TickInput
| Field | Type | Notes |
|---|---|---|
| up, down, left, right | boolean | Raw physical directions currently held — not facing-relative |
| buttons | `Record<ButtonName, boolean>` | Currently-held button names (`a b c x y z`), matching a `.cmd` file's own lowercase tokens |
Defined in: `src/input/types.ts` (`RawPlayerInput`, every field always present); `src/wasm/core-types.ts` (`TickInput`, the same shape with every field optional — the core's per-tick input contract). `src/input/tick-input-source.ts`'s `read()` returns a `[RawPlayerInput, RawPlayerInput]` pair, sent to the core as `{p1, p2}`.

## KeyboardPlayerBindings
| Field | Type | Notes |
|---|---|---|
| up, down, left, right | string | `KeyboardEvent.code` values (physical key, layout-independent) |
| buttons | `Record<ButtonName, string>` | `KeyboardEvent.code` per button name |
Defined in: `src/input/key-bindings.ts`; `DEFAULT_KEYBOARD_BINDINGS` gives Player 1 a WASD-based block and Player 2 an arrow-key-based block, spatially separate so two players can share one keyboard without a key ever bound to both.

## DrawCommand
| Field | Type | Notes |
|---|---|---|
| kind | `"sprite" \| "placeholder"` | Discriminant |
| canvasX, canvasY | number | The sprite's axis (pivot) point in canvas space |
| axisX, axisY | number | That point's pixel offset within the sprite image from its own top-left |
| width, height | number | Sprite/placeholder dimensions |
| pixels | `Uint8Array` | Only present for `kind: "sprite"` |
| flipH, flipV | boolean | Resolved mirroring — facing/frame-flip for a fighter, always `false` for a stage element (no axis metadata available, see `.vibe/decisions/004`) |
Defined in: `src/rendering/scene-composition.ts`
