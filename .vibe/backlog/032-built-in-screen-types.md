---
status: in_progress
---
# Built-in screen-type library

## Description
Roadmap decision `030`: refactor today's hard-wired versus flow into reusable screen types with parameters: title/main menu, character select (1 or N per side), stage select, match setup, VS, match, results, continue, victory/ending, options, controls, pause, move list. Each screen type takes its parameters from the mode and its appearance from the screenpack (item 030).

## Acceptance Criteria
- [ ] Each screen type runs from parameters, not from code specific to the versus flow
- [ ] Native and web builds expose the same screen types
- [ ] Keyboard and gamepad navigation work on every screen
