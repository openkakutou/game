---
status: todo
depends_on: [021]
---
# Move list screen from the pause menu

## Description
The pause menu opens the current character's move list. Content comes from the character's Ikemen GO movelist file (`[Files] movelist`, read by `character`), falling back to the commands of its `.cmd` when it has none. Appearance comes from the screenpack's move list screen.

Cross-repo: waits for character#073.

## Acceptance Criteria
- [ ] A character with an Ikemen GO movelist file shows it, glyphs included
- [ ] A character without one shows its commands from `.cmd`
- [ ] Each player sees the move list of their own character
