---
status: todo
depends_on: [071]
---
# Render OpenKakutou-exclusive 3D characters in a match

## Description
Roadmap decision `028`: a match may involve a 3D character (`.def` `model` key), against a sprite character or another 3D one. Render it in the native renderer (Go glTF loader + skinning on the existing `go-gl` path) and the web build, with facing = 180° rotation and palette = `KHR_materials_variants` variant. `engine` is unchanged: it still sees action numbers, durations and Clsn rectangles. Custom states (`ChangeAnim2`) using standard action numbers must display the opponent's own poses, whether sprite or 3D.

Blocked on `character` items 070 (parsing) and 071 (sample fixture), and on this repo's item 071 (OpenGL 3.3 core: skinning needs vertex shaders, decision 028). The native 3D renderer lives in `internal/gfx`, the only package importing go-gl.

## Acceptance Criteria
- [ ] A 3D vs sprite match plays, both sides rendered correctly
- [ ] Turning around rotates the 3D model; Clsn debug overlay stays aligned
- [ ] Get-hit and throw poses from custom states show the 3D character's own poses
- [ ] Frame budget benchmark (item 025) includes a 3D character case
- [ ] Per-character asset budget (triangles, texture size and count) set and written in `character`'s 3D extension spec, and a 3D vs 3D match with characters at that budget holds 60 fps (no frame over 16.6 ms) at 1080p on the reference low-end machine (Radeon Vega 10 Mobile integrated GPU, 2 GiB shared VRAM, decision 028)
