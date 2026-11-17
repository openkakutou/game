---
status: todo
---
# Render OpenKakutou-exclusive 3D characters in a match

## Description
Roadmap decision `028`: a match may involve a 3D character (`.def` `model` key), against a sprite character or another 3D one. Render it in the native renderer (Go glTF loader + skinning on the existing `go-gl` path) and the web build, with facing = 180° rotation and palette = `KHR_materials_variants` variant. `engine` is unchanged: it still sees action numbers, durations and Clsn rectangles. Custom states (`ChangeAnim2`) using standard action numbers must display the opponent's own poses, whether sprite or 3D.

Blocked on `character` items 070 (parsing) and 071 (sample fixture).

## Acceptance Criteria
- [ ] A 3D vs sprite match plays, both sides rendered correctly
- [ ] Turning around rotates the 3D model; Clsn debug overlay stays aligned
- [ ] Get-hit and throw poses from custom states show the 3D character's own poses
- [ ] Frame budget benchmark (item 025) includes a 3D character case
