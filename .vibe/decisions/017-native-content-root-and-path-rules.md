---
date: 2026-11-18
status: accepted
---
# Native content root and cross-platform path rules

**Context:** Backlog item 037 (roadmap decisions `031` and `033`): the native game must read the same content folder unchanged on Windows, macOS and Linux, with MUGEN/Ikemen GO content found as is whatever its file-name case.

**Decision:** The content root is the folder holding the executable, or the folder holding the `.app` when the executable sits in `<X>.app/Contents/MacOS/`; `-content` still overrides it. Every path the game builds goes through one resolver: `\` is a separator, `..` is resolved textually first and may not leave its base, each component matches its on-disk name exactly or else case-insensitively, and absolute paths and symbolic links are refused. A rejected reference becomes an unopenable path. A missing referenced file keeps its plain path, so the open error names it. A stage's `spr` and `bgmusic` are looked up beside the `.def`, then from the content root, in Ikemen GO's order (`src/stage.go`: `LoadFile("spr", []string{def, "", "data/"}, …)`), without `data/`, which is not part of this layout. A missing `chars/` or `stages/` folder is reported on the error screen, never created.

**Reason:** The resolver mirrors the `character` library's own case-insensitive lookup, so both layers agree. Resolving `..` textually before touching the disk stops a `..` hidden behind a missing folder from escaping. The content folder sits next to the program and is often read-only (inside an install directory or beside an `.app`), so reporting a missing folder is safer than creating one.

**Rejected alternatives:**
- *Fall back to a plain join on any resolution failure:* it reopened exactly the symlinked and escaping paths the resolver had refused.
- *Look up stage files beside the `.def` only:* root-relative community stages (`spr = stages/kfm.sff`) would load without sprites or music.
- *Create missing content folders:* it fails on read-only locations, and the item allowed reporting instead.
