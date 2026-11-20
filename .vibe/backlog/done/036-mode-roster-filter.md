---
status: done
depends_on: [031]
---
# Mode roster filter (character and stage visibility)

## Description
Roadmap decision `034`: visibility belongs to the mode. A mode may include or exclude characters and stages by folder name, separately for player selection and for CPU opponents (e.g. a boss excluded from selection but fought last in arcade). Default: everything selectable.

## Acceptance Criteria
- [ ] A character excluded from selection is not shown on the select screen but can still be a CPU opponent when the mode says so
- [ ] A filter naming a folder that is not installed is reported, not fatal

## Done (2026-11-20)
Native: `rules.roster` (select/opponents include/exclude, stages select), select screens over the visible entries, `OpponentPool()` for arcade (044). Reviewed, 2 findings fixed. Web: through the core flow (065).
