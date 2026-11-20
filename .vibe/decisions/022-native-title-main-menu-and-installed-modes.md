---
date: 2026-11-20
status: accepted
---
# Native title screen, main menu and installed modes

**Context:** Items 034 (title screen and main menu listing installed modes) and 035 (discover modes from `modes/`), native first.

**Decision:** The desktop game opens on a title screen ("press Enter or Start"), then a main menu listing every valid mode found in `modes/` (`*.json`/`*.jsonc`, case-insensitive), ordered by file name ignoring case like characters and stages, then Quit. An invalid mode file is left out with its error on stderr; the others still load. When `modes/` holds no valid Versus (id `versus`), the built-in copy is listed, as item 033 settled. Choosing a mode runs its flow; cancelling on its first screen, results' "back" with no select step, and the pause menu's Exit return to the main menu; cancel on the main menu returns to the title, cancel on the title quits. Quick versus skips all of this and still quits at match end. The Options entry is added with the first options screen (item 029), not as an empty placeholder now. The web half of 035 joins item 068 (build-time manifests).

**Reason:** Matches roadmap decision 030's runtime screens with the least new state; leaving a mode back to the menu is what Ikemen GO's pause Exit does for a menu-started match, and an empty Options screen would ship a dead end.

**Rejected alternatives:** MUGEN/Ikemen GO's title-is-the-menu layout (decision 030 names a title screen then a main menu; the screenpack, item 030, can restyle it later); ordering modes by their `name` (folder/file order is the rule already used for all other content, decision 034).
