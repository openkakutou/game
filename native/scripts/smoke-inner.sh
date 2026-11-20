#!/usr/bin/env bash
# Runs inside the container (see smoke.sh).
set -euo pipefail
out=dist/smoke; rm -rf "$out"; mkdir -p "$out"
Xvfb :99 -screen 0 1280x800x24 >/dev/null 2>&1 &
xvfb=$!; trap 'kill $xvfb 2>/dev/null || true' EXIT
export DISPLAY=:99 LIBGL_ALWAYS_SOFTWARE=1
sleep 1

# Every run uses a copy of the fixture: the game writes config/ files on first
# launch and when the player changes a setting.
content="$out/content"
cp -r testdata/content "$content"

# 1. Whole flow, driven by real key events.
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/run.log" 2>&1 &
pid=$!
sleep 2
shot() { import -window root "$out/$1.png"; }
key() { xdotool key --clearmodifiers "$@"; sleep 0.4; }
shot 0a-title
key Return; shot 0b-main-menu    # title -> main menu
key Return                       # Versus, the only mode of the fixture
shot 1-select-p1
key Return; shot 2-select-p2
key Right Return; shot 3-stage
key Return; shot 4-setup
key Return; shot 4b-versus       # start (cursor is on "Start"): the VS screen
key Return                       # skip the VS screen: the match
sleep 2; shot 5-match
xdotool keydown f; sleep 1; xdotool keyup f; shot 6-match-attack
key Escape; sleep 0.5; shot 6b-pause      # pause menu over the frozen match
sleep 1; shot 6c-pause-held
cmp -s "$out/6b-pause.png" "$out/6c-pause-held.png" || { echo "the match moved while paused"; exit 1; }
key Return; sleep 0.5                     # Continue
kill -0 $pid                     # still running = no crash
kill $pid; wait $pid 2>/dev/null || true

# Audio: the mixer opened on the dummy driver, the stage music loaded without
# a warning, and the fighters' PlaySnd reached the player (the fixture
# characters have no sound bank, so it logs the expected "no such sample").
grep -q '^audio: dummy$' "$out/run.log" || { echo "mixer did not open on the dummy driver:"; cat "$out/run.log"; exit 1; }
! grep -qE 'audio: (cannot|some|music|could not)' "$out/run.log" || { echo "audio warning during the match:"; cat "$out/run.log"; exit 1; }
grep -q 'audio: skipping sound 5,0' "$out/run.log" || { echo "PlaySnd never reached the audio player:"; cat "$out/run.log"; exit 1; }
! grep -q "built-in Versus mode" "$out/run.log" || { echo "the Versus mode was not read from modes/:"; cat "$out/run.log"; exit 1; }

# The stage's own sky sprite (#5A96E6) must be drawn, not the flat fallback backdrop.
sky=$(convert "$out/5-match.png" -format '%[hex:u.p{200,300}]' info:)
[ "$sky" = "5A96E6" ] || { echo "stage background layer missing: top-left pixel is $sky, want 5A96E6"; exit 1; }

# 1b. Player 2 set to CPU on the setup screen: the CPU must walk on its own.
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 &
pid=$!
sleep 2
key Return; key Return                     # title, Versus
key Return; key Right Return; key Return   # P1, P2, stage
key Up Right; shot 9-setup-cpu             # Player 2 row -> CPU
key Down Return                            # Start
key Return                                 # skip the VS screen
sleep 3; shot 10-match-cpu
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
cmp -s "$out/5-match.png" "$out/10-match-cpu.png" && { echo "CPU did not move"; exit 1; }

# 1c. Options: the first launch wrote the default settings files; Options >
# Audio, Music one step down, Back, Quit must save config/audio.json with it.
for f in gameplay audio video; do
  [ -f "$content/config/$f.json" ] || { echo "first launch did not write config/$f.json"; exit 1; }
done
grep -q '"music": 100' "$content/config/audio.json" || { echo "default audio.json is wrong:"; cat "$content/config/audio.json"; exit 1; }
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/options.log" 2>&1 &
pid=$!
sleep 2
key Return                       # title -> main menu
key Down Return; shot 13-options # Options
key Down Return; shot 14-options-audio
key Down Left; shot 15-options-audio-music
key Escape Escape                # leave Audio (saves), leave Options
key Down Return                  # Quit
wait $pid || { echo "the game did not quit cleanly from the main menu:"; cat "$out/options.log"; exit 1; }
grep -q '"music": 90' "$content/config/audio.json" || { echo "audio.json was not saved with music 90:"; cat "$content/config/audio.json"; exit 1; }
grep -q '"master": 100' "$content/config/audio.json" || { echo "audio.json lost master:"; cat "$content/config/audio.json"; exit 1; }

# 2. Deterministic in-program capture (the title screen) and error screen.
./dist/openkakutou-linux-amd64 -content "$content" -frames 5 -capture "$out/7-capture.png"
./dist/openkakutou-linux-amd64 -content /nonexistent -frames 5 -capture "$out/8-error.png"

# 3. Quick versus: two characters on the command line start the match with
# no menu (Ikemen GO's command line), so frame 30 already shows the fight.
./dist/openkakutou-linux-amd64 fighter rival -content "$content" -s arena -rounds 1 -p2.ai 1 -frames 30 -capture "$out/11-quick-versus.png"
cmp -s "$out/7-capture.png" "$out/11-quick-versus.png" && { echo "quick versus showed the menu"; exit 1; }

# 4. The HUD draws the content's lifebars/default (testdata, original art):
# P1's full life bar is its yellow front sprite (#FAD23C), not the built-in bars.
./dist/openkakutou-linux-amd64 fighter rival -content "$content" -s arena -width 960 -height 720 -frames 30 -capture "$out/12-lifebar.png"
bar=$(convert "$out/12-lifebar.png" -format '%[hex:u.p{150,40}]' info:)
[ "$bar" = "FAD23C" ] || { echo "lifebar not drawn: P1 bar pixel is $bar, want FAD23C"; exit 1; }

for f in "$out"/*.png; do
  colors=$(identify -format %k "$f")
  echo "$(basename "$f"): $colors colors"
  [ "$colors" -gt 2 ] || { echo "blank screenshot: $f"; exit 1; }
done
echo "smoke OK"
