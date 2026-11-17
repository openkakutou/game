---
status: todo
depends_on: [037, 038, 039, 040]
---
# Universal release: one portable folder for every desktop OS

## Description
Roadmap decision `033`: the release is one zip holding the Windows, macOS and Linux programs side by side at the root, plus `common/`, `lifebars/default`, `screenpacks/default`, the built-in modes and an empty `config/`, `chars/` and `stages/` (or the sample content the release already ships).

## Acceptance Criteria
- [ ] Unzipping and running the program of the current OS starts the game on every desktop OS
- [ ] The tree matches roadmap decisions `031` and `033`
