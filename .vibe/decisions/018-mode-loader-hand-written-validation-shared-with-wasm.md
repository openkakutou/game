---
date: 2026-11-19
status: accepted
---
# Mode loader: hand-written validation in one Go package, shared with the web through WASM (item 031)

**Context:** Roadmap decision `030` puts the mode format, its schema and its loader in `game`, as a Go package imported by the native game and compiled to WASM for the web game. The loader must name the file, field and reason of every error.

**Decision:** `native/internal/mode` parses JSONC itself (comments and trailing commas stripped, strings untouched) and validates the decoded JSON field by field, producing a `mode.Error{File, Field, Reason}` with a JSON path. `native/cmd/mode-wasm` exposes the same `Parse` to the web as `OpenKakutouMode.parse(file, text)`, smoke-tested under Node in the Native workflow. `mode.schema.json` is the published contract for editors and authors; a test fails when its value lists and the package's diverge, and the example was checked against it with a JSON Schema validator (ajv). The web bridge that downloads and calls this WASM is written with its first web consumer (item 033).

**Reason:** The standard library has no JSON Schema validator. Hand-written validation gives exact field paths and plain reasons, with no dependency in a WASM build where size matters. The schema stays the authors' reference.

**Rejected alternatives:** *A Go JSON Schema library validating against `mode.schema.json`:* one source of truth, but a dependency pulled into both builds, and its error messages are about schema keywords, not fields a mode author knows. *Decoding straight into structs with `DisallowUnknownFields`:* the decoder's errors do not give the full field path. *A separate repo for the format:* rejected by decision `030`, `game` is its only consumer.
