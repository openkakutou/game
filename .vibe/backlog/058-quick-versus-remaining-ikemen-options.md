---
status: todo
---
# Quick Versus: Remaining Ikemen GO Command-Line Options

## Description
Quick versus (decision `019`) supports Ikemen GO's core options; the rest are accepted with a "not supported yet" notice (`docs/command-line.md`). Support them as the capabilities they need land: `-p<n>.color` (palettes), `-p<n>.ai` levels (CPU difficulty, item 023), `-p<n>.input` (item 022), `-p<n>.lifeMax`/`.dizzyPoints`/`.guardPoints`, `-draws`, `-p3`+ and `-tmode1`/`-tmode2` (teams), `-r`/`-fight`/`-loadmotif` (screenpack, lifebars), `-log`, `-setvolume`, `-nojoy`, `-togglelifebars`, `-maxpowermode`, `-speed`.

🔒 MUST: behave as Ikemen GO (commit `54d4148a`, `main.lua:1023-1256`, roadmap decision `038`).

## Acceptance Criteria
- [ ] Each option listed above either works as in Ikemen GO or stays in the notice list with its reason in `docs/command-line.md`
- [ ] The smoke test starts a quick versus match using every supported option
