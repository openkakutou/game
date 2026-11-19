---
status: todo
---
# Stage Fills the Screen and Camera

## Description
In the native game, the Training stage covers only the right half of the screen. The left half is black, while Ikemen GO shows the whole stage (fidelity video, 2026-11-19). Draw the stage's layers over the whole view, and have the camera follow the fighters within the stage's camera bounds (`[Camera]` boundleft/boundright, `startx`, `zoffset` and `tension`), as Ikemen GO does (camera.go, commit `54d4148a`).

🔒 MUST: stages render as in Ikemen GO (roadmap decision `038`).

## Acceptance Criteria
- [ ] The Training stage fills the view at round start, as in Ikemen GO
- [ ] The camera follows the fighters and stops at the stage's bounds, as Ikemen GO computes it, with the source cited
- [ ] Checked in the real game with the fidelity video

## Notes
Found by the fidelity video after engine#062 (2026-11-19).
