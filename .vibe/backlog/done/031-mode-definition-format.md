---
status: done
---
# Mode definition format, schema and loader

## Description
Roadmap decision `030`: a mode is a JSONC file (flow of built-in screens + match rules) validated by a published JSON Schema. Write the format spec and schema, and a Go package that loads and validates a mode (native import, WASM for web). Rules cover players and characters per side, rounds, CPU opponents or ladder, win/lose conditions; flow lists built-in screen types with parameters. No scripting.

## Acceptance Criteria
- [x] Spec and JSON Schema published in the repo, with a complete example
- [x] Loader returns the validated mode or a clear error naming the file, field and reason
- [x] Same loader used by native (Go import) and web (WASM)

## Notes
The web bridge that downloads and calls the mode WASM lands with item 033, its first web consumer (decision 018).
