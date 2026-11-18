---
status: done
---
# Content root and cross-platform file rules

## Description
Roadmap decisions `031`, `033`: the game folder is the folder holding the program (on macOS, the folder holding the `.app`), with `chars/`, `stages/`, `lifebars/`, `screenpacks/`, `modes/`, `common/`, `config/`. Every reader applies the cross-platform rules: case-insensitive file names, `\` accepted as a separator, paths relative to the folder, UTF-8 JSON, no symbolic links; names the game writes are valid on Windows, macOS, Linux and FAT/exFAT.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO content found as-is, whatever the case of its file names.

## Acceptance Criteria
- [ ] The same folder runs unchanged from Windows, macOS and Linux programs placed at its root
- [ ] A content file referenced with the wrong case or with backslashes is found on Linux
- [ ] A missing content folder is created empty or reported, never a crash
