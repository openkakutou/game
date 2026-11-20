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
# The screenpack's menu audio: the built-in theme (Ogg Vorbis, so the mixer's
# decoder works), and the confirm (title) and move (select) sounds, each
# loaded once on first use, with no warning (checked above).
for loaded in "menu music loaded: sounds/theme.ogg" "menu sound loaded: sounds/confirm.ogg" "menu sound loaded: sounds/move.ogg"; do
  grep -q "audio: $loaded" "$out/run.log" || { echo "missing in the log: audio: $loaded"; cat "$out/run.log"; exit 1; }
done
! grep -q "built-in Versus mode" "$out/run.log" || { echo "the Versus mode was not read from modes/:"; cat "$out/run.log"; exit 1; }

# The stage's own sky sprite (#5A96E6) must be drawn, not the flat fallback backdrop.
sky=$(convert "$out/5-match.png" -format '%[hex:u.p{200,300}]' info:)
[ "$sky" = "5A96E6" ] || { echo "stage background layer missing: top-left pixel is $sky, want 5A96E6"; exit 1; }

# The title and the main menu are drawn from the built-in screenpack (the
# fixture has no screenpacks/), not the old flat backdrop: the logo's orange
# underline (#FF5C38, at the window centre, 640x231 in a 1280x800 display) is
# in both, and the gradient background and text give them hundreds of colours.
for shot_name in 0a-title 0b-main-menu; do
  logo=$(convert "$out/$shot_name.png" -format '%[hex:u.p{640,231}]' info:)
  [ "$logo" = "FF5C38" ] || { echo "$shot_name: the screenpack logo is missing: pixel is $logo, want FF5C38"; exit 1; }
  [ "$(identify -format %k "$out/$shot_name.png")" -gt 200 ] || { echo "$shot_name: too few colours for a screenpack screen"; exit 1; }
done

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

# 1d. Controls: Options > Controls > Player 1, the row of "a" waits for a key,
# U binds it; leaving saves config/controls.json (linux, P1 keyboard a = "U").
for f in controls; do
  [ -f "$content/config/$f.json" ] || { echo "first launch did not write config/$f.json"; exit 1; }
done
grep -q '"a": "F"' "$content/config/controls.json" || { echo "default controls.json is wrong:"; cat "$content/config/controls.json"; exit 1; }
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/controls.log" 2>&1 &
pid=$!
sleep 2
key Return                       # title -> main menu
key Down Return                  # Options
key Down Down Down Return; shot 16-options-controls
key Return; shot 17-controls-player1
key Down Down Down Down Return; shot 18-controls-waiting   # the row of "a"
key u; shot 19-controls-bound
key Escape Escape Escape         # leave Player 1 (nothing saved yet), Controls (saves), Options
key Down Return                  # Quit
wait $pid || { echo "the game did not quit cleanly after rebinding:"; cat "$out/controls.log"; exit 1; }
sed -n '/"linux"/,/"p2"/p' "$content/config/controls.json" | grep -q '"a": "U"' || { echo "controls.json was not saved with P1 a = U:"; cat "$content/config/controls.json"; exit 1; }
# A second launch loads and resolves it without a notice, then a match plays
# (the idle animation loops, so a screenshot cannot tell the attack from it:
# the binding itself is covered by the file check above and the unit tests).
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/rebound.log" 2>&1 &
pid=$!
sleep 2
key Return; key Return                     # title, Versus
key Return; key Right Return; key Return   # P1, P2, stage
key Return                                 # start
key Return                                 # skip the VS screen
sleep 2
xdotool keydown u; sleep 0.3; shot 20-rebound-new-key; xdotool keyup u
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
! grep -q 'controls.json' "$out/rebound.log" || { echo "the rebound controls.json was refused:"; cat "$out/rebound.log"; exit 1; }
sed -n '/"linux"/,/"p2"/p' "$content/config/controls.json" | grep -q '"a": "U"' || { echo "controls.json lost the binding"; exit 1; }

# 1e. Portraits: the fixture characters have none, so this run adds Kung Fu Man
# (testdata/reference, with the sprites 9000,0 and 9000,1). Hovering him draws his large
# portrait on the left: the pixel 342x288 is part of it (skin, not the
# background gradient the fixture fighter leaves there).
portraits="$out/content-portraits"
cp -r testdata/content "$portraits"
mkdir -p "$portraits/chars/kfm"
cp testdata/reference/chars/kfm/kfm.def testdata/reference/chars/kfm/kfm.sff "$portraits/chars/kfm/"
./dist/openkakutou-linux-amd64 -content "$portraits" -width 960 -height 720 >"$out/portraits.log" 2>&1 &
pid=$!
sleep 2
key Return; key Return                     # title, Versus: the select, on the fighter
shot 21-select-no-portrait
key Right; shot 22-select-portrait         # Kung Fu Man
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
before=$(convert "$out/21-select-no-portrait.png" -format '%[hex:u.p{342,288}]' info:)
after=$(convert "$out/22-select-portrait.png" -format '%[hex:u.p{342,288}]' info:)
[ "$before" != "$after" ] || { echo "Kung Fu Man's large portrait is not drawn: pixel 342x288 is still $after"; exit 1; }

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

# Options, controls, match setup and VS are drawn from the screenpack too: its
# gradient background (#1A1826 at 200x100, not the old flat #121420) and the
# hundreds of colours of anti-aliased text; the pause menu, drawn over the
# frozen match, has the same colour count (the old one had about 26).
for shot_name in 1-select-p1 2-select-p2 3-stage 13-options 14-options-audio 15-options-audio-music 16-options-controls 17-controls-player1 18-controls-waiting 19-controls-bound 4-setup 4b-versus 9-setup-cpu; do
  bg=$(convert "$out/$shot_name.png" -format '%[hex:u.p{200,100}]' info:)
  [ "$bg" = "1A1826" ] || { echo "$shot_name: the screenpack background is missing: pixel is $bg, want 1A1826"; exit 1; }
  [ "$(identify -format %k "$out/$shot_name.png")" -gt 200 ] || { echo "$shot_name: too few colours for a screenpack screen"; exit 1; }
done
[ "$(identify -format %k "$out/6b-pause.png")" -gt 200 ] || { echo "6b-pause: too few colours for the screenpack pause menu"; exit 1; }

for f in "$out"/*.png; do
  colors=$(identify -format %k "$f")
  echo "$(basename "$f"): $colors colors"
  [ "$colors" -gt 2 ] || { echo "blank screenshot: $f"; exit 1; }
done
echo "smoke OK"
