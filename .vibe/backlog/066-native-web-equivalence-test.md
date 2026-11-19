---
status: todo
depends_on: [064]
---
# Native/web equivalence test and docs

## Description
Step 5 of item 060. A test replays the same scripted match through the core natively and through core.wasm under Node and compares the state every tick. `CLAUDE.md` and `docs/` describe the core/adapters split (with a diagram).

## Acceptance Criteria
- [ ] The equivalence test runs in CI and fails on any per-tick difference
- [ ] `CLAUDE.md` and docs describe the core and the per-platform adapters
