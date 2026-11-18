---
status: todo
depends_on: [051]
---
# Side-by-side fidelity videos: Ikemen GO vs OpenKakutou

## Description
The showcase site (roadmap#024) shows how faithful OpenKakutou is with videos: the same match, driven by the same scripted inputs, played in Ikemen GO and in `game`, shown side by side. Build the tool that produces them from the differential replay (051): step both engines frame by frame on the same replay, capture every frame on both sides, and encode one video with the two images side by side, labelled "Ikemen GO" and "OpenKakutou", with the frame counter and the frames identical before the first divergence. Optional third panel or overlay: the per-pixel difference.

Recorded with no wall clock (frame-stepped), so the video shows exactly what the comparison compared; sound comes from one engine only, or none.

**No CPU**: scripted or recorded inputs only, like 051.

Only content that works in Ikemen GO and whose license allows redistribution (the videos are published); credits are written into the video's caption file.

🔒 MUST: MUGEN 1.0/1.1 and Ikemen GO compatibility, verified through Ikemen GO.

## Acceptance Criteria
- [ ] One command turns a recorded replay into a side-by-side video (web-friendly format, 60 fps, both engines at the same internal resolution)
- [ ] The video shows both engine names, the frame counter, and the frames identical before the first divergence
- [ ] A caption file lists characters, stage, lifebar and their authors and licenses
- [ ] At least one video produced on redistributable content, ready for roadmap#024

## Notes
Cross-repo: needs engine#036 (replay, seed, Ikemen GO trace) through 051. Consumed by roadmap#024 (showcase site). Encoding may use an external encoder (e.g. ffmpeg) as a tool, not as a runtime dependency of `game`.
