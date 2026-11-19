---
date: 2026-11-19
status: accepted
---
# Quick versus follows Ikemen GO's command line

**Context:** The Product Owner asked for MUGEN's and Ikemen GO's quick versus: the program, given characters on its command line, starts one match without menus. "Versus" is the name of the built-in mode; quick versus is this program capability, not a mode.

**Decision:** The native program parses its arguments with Ikemen GO's algorithm (`processCommandLine`, `src/main.go:241-361`, commit `54d4148a`: two leading names are `-p1`/`-p2`, a flag takes the next argument unless it is another flag, a numeric value may be negative, PowerShell-split `-p1 .ai` is joined) and plays quick versus as `main.f_commandLine` does (`main.lua:1023-1256`): `-rounds` is the rounds to win, `-time -1` means no limit, an unknown stage falls back to the default stage with a notice, an unknown character stops with an error, the game quits after the match, `-p<n>.life`/`.power` apply to round 1 only (`system.go:4417-4430`). Characters and stages are resolved as `resolveCharDef`/`resolveStageDef` (`system.go:5370-5497`), with `common/` in the place of `data/` (roadmap decision `031`) and case-insensitive names. Options the game cannot honour yet are accepted and named in a notice; the match still starts. The `flag` package is dropped: its own parsing cannot accept positional names before options.

**Reason:** A launcher written for MUGEN or Ikemen GO (such as the Product Owner's own quick-versus launcher) then works unchanged, and the behaviour is checked against Ikemen GO's source, the org's reference (roadmap decision `038`).

**Rejected alternatives:** *OpenKakutou-specific flags (`--p1=...`):* existing launchers would break. *Refusing unsupported Ikemen GO options:* a launcher passing `-p1.color` would not start at all; a notice keeps the match playable and says what was ignored. *Mapping the AI level onto the CPU:* the CPU has a single behaviour; the level is reported as unused instead of faking a difference.
