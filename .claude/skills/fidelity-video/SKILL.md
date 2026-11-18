---
name: fidelity-video
description: Make a side-by-side demo video of the same match in Ikemen GO (reference build) and in OpenKakutou, driven by the same scripted P1 key presses, to see how far OpenKakutou is from Ikemen GO. Use when the user asks for a fidelity, comparison or demo video, or gives two characters (and optionally a stage) to film.
---

# Fidelity video: Ikemen GO vs OpenKakutou

Films one match twice, with the same key presses, and puts the two recordings side by side:
- left: Ikemen GO's Linux release (`nightly` by default, which is the reference commit of roadmap decision 041; check it with `gh release view nightly -R ikemen-engine/Ikemen-GO`);
- right: OpenKakutou's native game (`game/native`), built on the local `engine` and `character` HEAD commits.

Both run headless in Docker (Xvfb, software OpenGL), driven by `xdotool`, recorded by `ffmpeg x11grab` at 60 fps. Real time, not frame-stepped: a first look, not the frame-exact comparison of game#051 (game#052 makes that version).

## Inputs

- Two character `.def` files (P1 and P2) and a stage `.def`. Take them from the user's Ikemen GO collection (`~/retrogaming/Ikemen/`): any content that works in Ikemen GO; only the video is shared, never the content (Product Owner, 2026-11-18). Credit the authors when the video is published.
- No stage given: use `~/retrogaming/Ikemen/quick-versus-bootstrap/stages/Training/Training.def`.
- Before filming, check the pair loads in Ikemen GO: a character that fails there shows a GTK error dialog in the left half. Pick another one and say so.

## Run

```
game/.claude/skills/fidelity-video/scripts/run.sh \
  --p1 <P1.def> --p2 <P2.def> --stage <stage.def> --out <video.mp4> \
  [--work <dir>] [--inputs <script.sh>] [--ik-wait 17] [--seconds 16] \
  [--ikemen-tag nightly] [--engine <ref>] [--character <ref>]
```

Put `--work` and `--out` in the scratchpad; reuse the same `--work` between runs (the Ikemen GO download, its config and the Go caches are kept). A first run takes about 8 minutes, later ones about 2.

- `--ik-wait`: seconds from Ikemen GO's launch to "Fight!". It depends on the characters' intros: check the left half of the preview, and raise it if the first moves happen during "Get ready".
- `--inputs`: a bash script of `xdotool` key presses, P1 only, same keys in both engines (W A S D move; F G H = a b c; R T Y = x y z). The default (`scripts/inputs.sh`) walks, uses a, b, x, a crouching c, walks back, jumps, then tries D-DF-F+a and F-D-DF+x, facing right.

## Check, then hand over

1. Read the preview image (`<out>.png`) and look at a contact sheet of both halves (`ffmpeg -i <out> -vf "fps=1,scale=320:-1,tile=4x4" -frames:v 1 sheet.png` inside the recording image) before saying anything about the result.
2. If the OpenKakutou half shows "Something went wrong", quote the error from `<work>/openkakutou.log` or the screen and name the backlog item that covers it (engine 039 VarSet form, 041 expression grammar, 028 triggers…); if none does, say so and offer one.
3. Send the video to the user (SendUserFile), with the absolute path, and say in plain words what each half shows and where OpenKakutou stops.
