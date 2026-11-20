---
status: todo
depends_on: [064, 067]
---
# Web screens through the core flow

## Description
Step 4 of item 060. Character select, stage select, match setup, results and rematch on the web follow the core's flow (the same state machine as native), so complete-game items 031+ are built once. TypeScript keeps only drawing those screens and reading input.

## Acceptance Criteria
- [ ] `src/selection`, `src/setup`, `src/result` and `src/roster` hold no navigation or rule logic
- [ ] The web and native builds go through the same screens in the same order for the same inputs

## Notes
- Item 032 (2026-11-20) made the native flow walk a mode's steps (ADR 021), VS screen included: the web gets the same screen types by following that flow here.
