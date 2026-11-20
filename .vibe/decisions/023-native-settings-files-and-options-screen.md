---
date: 2026-11-20
status: accepted
---
# Native settings files and options screen

**Context:** Backlog item 029, native first: settings split by concern in `config/` inside the game folder (roadmap decisions 029, 032, 033), an options screen, Options entries in the main menu and the pause menu.

**Decision:**
- Files: `config/gameplay.json` (`rounds`, `timeLimitSeconds`), `config/audio.json` (`master`, `music`, `effects`, each 0–100), `config/video.json` (one section per platform, `windows`/`macos`/`linux`/`android`/`web`, each `fullscreen`, `vsync`, `windowWidth`, `windowHeight`; the running platform reads and writes its own section and keeps the others untouched). Controls (`controls.json`, item 022) and appearance (`appearance.json`, item 041, already read for the lifebar) are not touched here.
- Each concern loads independently: a missing file means defaults silently (and is written on first launch, creating `config/`); an unreadable or corrupt file means defaults for that concern only, a notice on stderr and on the options screen, and the file is not overwritten until the player changes that concern. Unknown fields are kept when the file is rewritten.
- Defaults: rounds 3, time 99 s (the Versus file's values), volumes 100, fullscreen off, vsync on, window 960×720 (today's defaults). `-width`/`-height` on the command line still override the window size for that run without saving.
- Gameplay settings are the preselected values of a mode's `matchSetup` screen; a mode without `matchSetup` keeps its own rules (fixed-rule modes such as a future training or arcade decide for themselves).
- Options screen: Gameplay, Audio, Video, Back; each a list of rows changed with left/right, saved when leaving the sub-screen. Audio applies at once (and when a paused match resumes), video at once (fullscreen, vsync, window size), gameplay on the next match setup. Reached from the main menu (between the modes and Quit) and from the pause menu (before Exit).
- Reading and decoding live in the core (`fs.FS`, no `os`); writing goes through a function the host injects, so the core stays file-system free. The web half (browser storage, one key per concern) is item 069.

**Reason:** Follows the roadmap's file split and location; gameplay-as-setup-defaults keeps mode rules meaningful for modes that fix them, and keeps Versus unchanged for a player who never opens Options.

**Rejected alternatives:** Gameplay settings overriding every mode's rules (a training or arcade mode could not fix its own rules); one `settings.json` (against roadmap decision 029); writing corrupt files back with defaults at startup (would silently destroy a hand-edited file with one typo).
