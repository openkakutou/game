# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Fixed
- In the desktop game, a jumping fighter is drawn in the air instead of sinking below the ground.
- In the desktop game's Appearance options, the sample lifebar no longer covers the page heading.
- In the desktop game, a gamepad's B attack or the Backspace key no longer opens the pause menu during a match: only Escape and the gamepad's Start do.

### Added
- The round banner follows the round: "Round N" during the intros, "Fight!" as the fight starts, the outcome (KO, Time over...) as soon as the round is decided.
- Rounds play as in Ikemen GO (engine v2.46.0): the fighters start in their intro (kfm kicks its wood pieces) without control, the fight starts once the intros are done, and after a KO the round goes on, the inputs pause, the winner plays its win pose and the loser its lose pose before the next round. Training still fights from the first tick.
- The desktop game draws a character with its first .act palette, as Ikemen GO does by default: in an SFF v1 sheet it replaces the first palette, sprites with their own palette keep it (sff v0.9.0).
- Engine v2.44.0 and character v0.21.0: many more community characters load and play without stopping the match (a run over 373 corpus characters went from 22 to over 90 playing 900 random ticks cleanly).
- The desktop game draws explods (engine v2.42.0), the effects characters make with the Explod controller, behind or over the fighters by their sprite priority, from the character's sprites or the common fightfx.
- The desktop game draws helpers (engine v2.41.0): characters that throw their fireballs from helpers, such as Ryu (SVC), now play them; checked in the game with that Ryu's hadouken hitting kfm.
- The desktop game draws projectiles (engine v2.37.0) with their thrower's animations, flying across the stage until they hit or leave.
- The desktop game shows hit sparks where hits land and plays the hit and guard sounds, from the game's own common effects and sounds (or a character's own when it asks for them); additive animation frames now blend as in Ikemen GO.
- In the desktop game, fighters guard: holding back blocks attacks that can be blocked standing, crouching or in the air (low attacks need a crouching guard), with chip damage as in Ikemen GO; the Training dummy's Guard setting now really blocks (engine v2.35.0).
- In the desktop game, attacks that start before they reach (sweeps, long-reaching moves) now hit, knocked-down fighters fall, lie down and get up by themselves, as in Ikemen GO (engine v2.34.0).
- In the desktop game, fighters turn to face an opponent who gets behind them, hits push the opponent back and drive their falls, and characters' common states can make them unhittable, change the damage they take or block walking and jumping, as in Ikemen GO (engine v2.31.0).
- In the desktop game, each character uses its own life, gravity and ground friction from its files, as in Ikemen GO; the life bars measure its own life, and -p1.life/-p2.life are capped by it.
- Fighters can no longer walk off screen: as in Ikemen GO, the camera is part of the match and keeps them between the screen edges, with the stage's own margins.
- In the desktop game, a CPU-driven character with its own AI, such as Ikemen GO's Kung Fu Man (.zss version), now plays with that AI.
- The desktop game's CPU plays as Ikemen GO's: it presses random directions and buttons, more often at a higher level, and characters with their own AI get the level they read. The level is the new Difficulty option (1 to 8, default 4), or -ailevel and -p1.ai/-p2.ai on the command line.
- The desktop game's character select has a random cell after the last character, when the screenpack draws one: it picks a character at random.
- In the desktop game, screenpack panels and menu cursors have rounded corners, tiled backgrounds scroll, and menu music loops from its loop point, as the screenpack format describes.
- The desktop game accepts Ikemen GO's -setvolume, -nojoy, -togglelifebars, -maxpowermode, -fight (a lifebar file) and -r (a screenpack) command-line options for the current run.
- In the desktop game, a lifebar's names and timer are printed in the lifebar's own MUGEN / Ikemen GO bitmap fonts (.fnt and .def), in their colors; a font that cannot be read is reported and replaced by the built-in font.
- In the desktop game, a character's sound on a numbered channel cuts its previous sound on that channel, low-priority sounds wait for the channel to be free, and up to 32 effects play at once, as in Ikemen GO.
- The desktop game ships the common states characters rely on (Ikemen GO's, MIT-licensed) and plays the common hit and guard sounds from the game's own common sound file.
- The game ships its own common hit sparks, guard sparks and common sounds (CC0) in the common folder, covering every common effect and sound number used by the 436 characters of the test corpus; they are generated by a script kept in the repository.
- The desktop game has a Training mode: you against a dummy with no time limit and life refilled after a second without being hit, plus pause entries to reset positions and set the dummy (stand, crouch, jump, guard, CPU); guard only holds back until the engine supports blocking. Game modes can add these entries to the pause menu.
- The desktop game has an Arcade mode: pick a character and fight a ladder of random CPU opponents on random stages, with a continue countdown after a loss (Yes fights the same opponent again) and a victory screen after the last one; a mode can name a final boss with `opponents.final`.
- Game modes can choose which characters and stages their select screens show, and which characters a CPU opponent can be (for example a boss hidden from selection); a name that matches no installed folder is reported without stopping the game.
- The desktop game's pause menu has a Move List: the paused player's character's Ikemen GO move list with button icons, or its commands when it has none; left and right switch to the other character.
- The desktop game has a Watch mode: pick two characters and a stage, and watch them fight each other under CPU control.
- The desktop game lists every screenpack in the screenpacks folder and every lifebar in the lifebars folder; an Appearance page in the options picks them with a live preview, saved in the config folder, and a choice whose folder was removed falls back to the default with a message.
- The desktop game's title screen, main menu, options and controls, pause menu, character and stage select, match setup, VS and results screens are now drawn from the OpenKakutou screenpack (background, fonts, menu cursor, value colors) instead of plain text; the character select is a grid showing each character's own portraits and display name; menus play the screenpack's sounds and music and fade in.
- In the desktop game, each player can rebind their keyboard keys and gamepad buttons from Options > Controls; a key already in use is never taken silently (swap or cancel), and the bindings are saved in the game folder's config folder.
- The desktop game has an Options screen, from the main menu and the pause menu, for match rounds and time, sound volumes (master, music, effects) and display (fullscreen, vsync, window size); each kind of setting is saved in its own file in the game folder's config folder, and a broken file only resets its own settings, with a message.
- The desktop game opens on a title screen, then a main menu listing every game mode found in the modes folder, plus Quit; a broken mode file is left out with a message saying why, and leaving a mode (or its error screen) returns to the main menu.
- The desktop game reads its Versus mode from the modes folder of the game folder, so its screens and default rules can be changed by editing that file; a missing or broken file falls back to the built-in Versus with a message saying why.
- The desktop game now shows a VS screen with both chosen fighters before each match, and its screens follow the Versus mode's definition (preselected rounds and time come from it).
- In the desktop game, characters and stages dropped into their folders are listed in alphabetical order regardless of capital letters.

### Changed
- The web game now plays matches with the same engine as the desktop game, so a match plays the same on both: the CPU opponent, the camera that follows the fighters across the stage and the round and match results come from it.

## [0.26.0] - 2026-11-19

### Added
- Pause menu in matches, native and web: Escape or a gamepad's Start freezes the match and its sound and opens PAUSE with Continue, Restart and Character Change, plus Exit in the native game, as in Ikemen GO. Escape or Start again continues.

### Changed
- In the native game, Escape during a match now opens the pause menu instead of leaving the match at once; in a quick versus match, Exit in that menu quits.

## [0.25.0] - 2026-11-19

### Added
- The native match HUD draws a MUGEN / Ikemen GO lifebar: drop one in `lifebars/<name>/` and select it with `{"lifebar": "<name>"}` in `config/appearance.json`, or it uses `lifebars/default`. The life and power bars, their trailing fill, the names and the timer are placed as Ikemen GO places them; a broken lifebar folder is skipped with a notice, and with no lifebar at all the built-in bars stay. Faces, round and win icons, combo counts and the lifebar's own fonts are not drawn yet.

## [0.24.0] - 2026-11-19

### Added
- Quick versus: the native game takes Ikemen GO's command line, so `openkakutou kfm kfm -s Training -rounds 2` starts that match straight away, with no menu, and quits when it ends, as MUGEN and Ikemen GO do. Characters and stages are found as Ikemen GO finds them; player 1 or 2 can be given to the CPU, and their life and power set for round 1. Options not supported yet are named in a notice and the match still starts. `-h` lists them; `docs/command-line.md` describes them.

### Changed
- The built-in mode is now called Versus (`versus.jsonc`); quick versus names the command-line launch only. The window title is OpenKakutou.

## [0.23.0] - 2026-11-19

### Added
- Game modes can now be described as data: a mode file lists the screens a player goes through and the match rules (sides, rounds, time limit, opponents, what follows a win or a loss). The format is documented with a published JSON Schema and a Quick Versus example, and a mode with a mistake is refused with a message naming the file, the field and the reason. The same loader runs in the native game and, built to WebAssembly, in the web game.

## [0.22.1] - 2026-11-19

### Fixed
- A `.cns` character no longer loses control once it walks: Kung Fu Man can walk, stop and still punch and kick in the native game (engine v2.22.2, character v0.17.0).

## [0.22.0] - 2026-11-19

### Added
- The native game now follows each stage's own camera settings: the camera pushes at the stage's tension and starts at its start position, background layers are drawn at their declared scale, and the fighters stand on the stage's resolved ground line.

## [0.21.0] - 2026-11-19

### Fixed
- In the native game, a stage fills the whole screen instead of only its right half (e.g. Training), and the camera follows the fighters within the stage's bounds, as in Ikemen GO.

## [0.20.0] - 2026-11-19

### Changed
- The native game uses the latest engine. Fighters have control when a round starts, Kung Fu Man walks and jumps when the player holds forward or up, and a held button counts as one press, as in Ikemen GO.

## [0.19.0] - 2026-11-19

### Changed
- The native game uses the latest engine: real characters such as Kung Fu Man now get through their first ticks (constants, animation and opponent triggers, the full expression grammar).

## [0.18.0] - 2026-11-18

### Added
- The native game loads every state file of a character as Ikemen GO does, `.cns` and `.zss`, including its common states, which are searched in the character folder and then in the game's `common/` folder. A common file found nowhere is reported instead of stopping the load.

## [0.17.0] - 2026-11-18

### Added

- The native game now finds its characters and stages in the folder that holds the program (on macOS, the folder holding the app), so the same unzipped folder plays as is on Windows, macOS and Linux. Community content works whatever the letter case of its file names or its `\` separators, a stage's sprites and music are also found from the game folder as in Ikemen GO, symbolic links and paths leaving the game folder are refused, and a missing characters or stages folder is named on the error screen instead of crashing.

## [0.16.0] - 2026-10-30

### Added

- In the native game, the stage's music now plays during a match, looping across rounds and on the result screen, and fighters' sound effects play when they trigger, like the web version. If there is no sound device or a sound file cannot be read, the game stays silent and keeps running.

## [0.15.0] - 2026-10-30
### Added

- In the native game, a match now shows the stage's background art behind and in front of the fighters, including animated elements and slower-scrolling distant layers, like the web version.

## [0.14.0] - 2026-10-30

### Added

- In the native game, a connected gamepad now drives the menus and the match: the first pad is player 1, the second player 2, and a player without a pad keeps the keyboard, also after an unplug.

## [0.13.0] - 2026-10-16

### Changed

- Moved to the redesigned shared look (web-ui-kit 0.17, "Studio": dark by default, new colour tokens, 14 px text and 34 px controls).

### Added

- Native desktop game for Linux and Windows (a Mac build is set up in CI but not yet verified): a standalone program with its own character selection, stage selection, match setup (rounds and time limit), live match with lifebars, power bars, round wins and timer, and a result screen with rematch. Two players share one keyboard with the same keys as the web version. Characters and stages are read from a folder given on the command line. No sound, gamepad support or stage background art yet.
- Native desktop game: a new "Player 2" choice on the match setup screen (Human or CPU, Human by default) lets you play against a computer opponent that walks toward you and attacks now and then, same as the web version. A rematch keeps the choice.

## [0.12.0] - 2026-09-27

### Added

- Match audio (web build): a stage's background music now plays on loop for the whole match, and each fighter's hit/voice/taunt sound effects now play the moment they're triggered during combat. A stage with no background music, or a character sound that can't be matched, plays silently instead of causing an error or interrupting the match. Switching away from the browser tab and back doesn't cause audio to glitch or pile up. The native (Windows/Mac/Linux/Android) build does not yet play audio — no native build of this app exists in this repo yet, tracked as this item staying open.

## [0.11.0] - 2026-09-25

### Added

- Web release packaging: the production build now publishes automatically to GitHub Pages on every update to the main line, after the full test suite, lint, and a build all pass. If a character, stage, or match engine module fails to load at runtime (a missing or incompatible file), the affected screen now shows a clear message instead of a blank page or a silent failure.

## [0.10.0] - 2026-09-25

### Added

- Round and match results: when a round ends, a result panel shows the round winner (or a draw) and automatically continues to the next round after a short countdown. When the match itself ends, a result panel shows the overall winner (or a draw) with two choices: "Rematch" (instantly replays the match with the same characters, stage, and settings) or "Back to select" (returns to character selection). An unexpected or ambiguous match outcome always resolves to a clear result instead of leaving the screen stuck.
- Single-player mode: the match setup screen now has a "Player 2 Control" choice (Human or CPU), defaulting to Human. Choosing CPU hands player 2 to a minimal computer-controlled opponent — a first pass, not a full AI — that moves toward its opponent and occasionally attacks, driven through the exact same input path as a real player. The Controls section reflects the choice, showing "controlled automatically" for a CPU-driven player 2 instead of keyboard bindings.

## [0.9.0] - 2026-09-24

### Added

- In-match HUD: both players now see a live lifebar and power/meter bar during a match, plus a round display showing the current round number, each player's round wins, and the configured best-of. Everything updates in sync with the match as it plays. If the combat engine ever reports match data the HUD can't understand, the HUD shows a clear "could not update" message instead of freezing or crashing the match itself.

## [0.8.0] - 2026-09-17

### Added

- Automated screenshot-comparison tests now guard the roster (character) selection screen and the stage selection screen against unintended visual regressions, run in CI on every change and failing the build if a real rendering difference is introduced.

## [0.7.0] - 2026-09-09

### Added

- Localization (i18n): the character/roster selection, stage selection, and match setup screens are now available in English and French, with a language switcher in the toolbar that changes the displayed language instantly, without a page reload. The chosen language is detected from the browser by default and remembered the next time the app loads. Switching language never loses an in-progress pick (character, stage, round count, time limit). The in-match HUD and the not-yet-built round/match result screen are not covered by this change.

## [0.6.1] - 2026-09-08

### Fixed

- Fixed the production build failing at its type-checking step because of a test file's overly loose callback typing, unrelated to any actual app behavior.

## [0.6.0] - 2026-09-07

### Added

- Player input: both players' actions during a match are now read live and actually drive it. Each player has their own default keyboard controls (Player 1: WASD + F/G/H/R/T/Y, Player 2: arrow keys + J/K/L/I/O/P), shown on the match setup screen's new Controls section. A connected gamepad is detected and used automatically for whichever player it's assigned to, without requiring one to be present; if it disconnects mid-match, that player falls back to their keyboard controls, shown live via an in-match status line. Each character's own command file is now read and used, so recognized moves actually resolve during a match.

## [0.5.0] - 2026-09-07

### Added

- Match rendering: once both players' characters, a stage, and the match setup are all confirmed, the match now actually starts and renders — both characters' live sprites (correct position and current animation frame, driven by the combat engine) composited over the chosen stage's background, kept in sync with the simulation as it runs. A sprite that can't be resolved shows a clear placeholder instead of a blank gap or a crash.

## [0.4.0] - 2026-08-29

### Added

- The match setup screen: after picking characters and a stage, players choose the round count (1, 3, or 5 rounds) and the per-round time limit (60s, 99s, or unlimited) before starting the match. Both choices are required before continuing, and switching one choice never resets the other.

## [0.3.0] - 2026-08-29

### Added

- The stage selection screen: after both players pick a character, stages configured in a deployable stage list are validated and displayed with their name and a preview; a stage that fails to load shows a clear error card instead of crashing or being silently dropped. Players pick one shared stage together and can't continue until a stage is chosen.

## [0.2.0] - 2026-08-16

### Added

- The character roster discovery/selection screen: characters configured in a deployable roster manifest are validated and displayed with their name and a portrait; a character that fails to load shows a clear error card instead of crashing or being silently dropped. Player 1 and Player 2 each pick a character independently, including picking the same character for a mirror match, and can't continue until both have picked.

[Unreleased]: https://github.com/openkakutou/mode-quick-versus/compare/v0.26.0...HEAD
[0.26.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.25.0...v0.26.0
[0.25.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.24.0...v0.25.0
[0.24.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.23.0...v0.24.0
[0.23.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.22.1...v0.23.0
[0.22.1]: https://github.com/openkakutou/mode-quick-versus/compare/v0.22.0...v0.22.1
[0.22.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.21.0...v0.22.0
[0.21.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.20.0...v0.21.0
[0.20.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.19.0...v0.20.0
[0.19.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.18.0...v0.19.0
[0.18.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.17.0...v0.18.0
[0.17.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.16.0...v0.17.0
[0.16.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.15.0...v0.16.0
[0.15.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.14.0...v0.15.0
[0.14.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.13.0...v0.14.0
[0.13.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.12.0...v0.13.0
[0.12.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.11.0...v0.12.0
[0.11.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.10.0...v0.11.0
[0.10.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.9.0...v0.10.0
[0.9.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.8.0...v0.9.0
[0.8.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.7.0...v0.8.0
[0.7.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/openkakutou/mode-quick-versus/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/openkakutou/mode-quick-versus/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/openkakutou/mode-quick-versus/releases/tag/v0.2.0
