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
key Down Down Return             # Versus: Arcade, Training, Versus, Watch in file-name order
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
! grep -q "built-in Watch mode" "$out/run.log" || { echo "the Watch mode was not read from modes/:"; cat "$out/run.log"; exit 1; }

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
key Return; key Down Down Return           # title, Versus (after Arcade, Training)
key Return; key Right Return; key Return   # P1, P2, stage
key Up Right; shot 9-setup-cpu             # Player 2 row -> CPU
key Down Return                            # Start
key Return                                 # skip the VS screen
sleep 3; shot 10-match-cpu
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
cmp -s "$out/5-match.png" "$out/10-match-cpu.png" && { echo "CPU did not move"; exit 1; }

# 1b2. Watch: the second mode of the main menu. The player picks both
# fighters and the stage, and the match runs with nobody pressing a key: two
# captures a few seconds apart differ. That both sides are CPU-driven is
# checked by the game tests (TestWatchMode_...): comparing with the idle
# match capture was flaky under load (same frame when the CPUs had not moved
# yet).
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/watch.log" 2>&1 &
pid=$!
sleep 2
key Return; key Down Down Down Return      # title, Watch
key Return; key Right Return; key Return   # P1, P2, stage
key Return                                 # Start (no Player 2 row)
key Return                                 # skip the VS screen
sleep 2; shot 28-watch-match-a
sleep 3; shot 29-watch-match-b
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
cmp -s "$out/28-watch-match-a.png" "$out/29-watch-match-b.png" && { echo "the two CPUs did not move in Watch"; exit 1; }
! grep -q "built-in Watch mode" "$out/watch.log" || { echo "Watch was not read from modes/:"; cat "$out/watch.log"; exit 1; }

# 1b3. Arcade: the first mode of the main menu (Arcade, Training, Versus, Watch in
# file-name order). The player picks, the VS screen shows the first opponent
# (the fixture has two characters, so the ladder is the rival alone), a key
# skips it and the match starts.
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/arcade.log" 2>&1 &
pid=$!
sleep 2
key Return; key Return                     # title, Arcade
shot 30-arcade-select
key Return; shot 31-arcade-versus          # fighter picked: no stage select, no setup, the VS screen
key Return                                 # skip the VS screen
sleep 2; shot 32-arcade-match
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
! grep -q "built-in Arcade mode" "$out/arcade.log" || { echo "the Arcade mode was not read from modes/:"; cat "$out/arcade.log"; exit 1; }
sky=$(convert "$out/32-arcade-match.png" -format '%[hex:u.p{200,300}]' info:)
[ "$sky" = "5A96E6" ] || { echo "the arcade match did not start on the stage: top-left pixel is $sky, want 5A96E6"; exit 1; }

# 1b4. Training: the second mode of the main menu. Both fighters and the stage
# are picked, the match starts at once (no setup, no VS screen) and runs (an
# attack changes the picture); the
# pause menu then offers Reset Positions and Dummy after Restart (the capture
# is read by hand: Continue, Restart, Reset Positions, Dummy: Stand, ...).
./dist/openkakutou-linux-amd64 -content "$content" -width 960 -height 720 >"$out/training.log" 2>&1 &
pid=$!
sleep 2
key Return; key Down Return                # title, Training
key Return; key Right Return; key Return   # P1, P2 (the dummy), stage
sleep 2; shot 33-training-match
xdotool keydown f; sleep 1; xdotool keyup f; shot 34-training-match-attack
cmp -s "$out/33-training-match.png" "$out/34-training-match-attack.png" && { echo "the Training match did not run"; exit 1; }
key Escape; sleep 0.5; shot 35-training-pause
[ "$(identify -format %k "$out/35-training-pause.png")" -gt 200 ] || { echo "35-training-pause: too few colours for the pause menu"; exit 1; }
key Down Down Return; sleep 0.5            # Reset Positions: closes the menu
shot 36-training-reset
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
! grep -q "built-in Training mode" "$out/training.log" || { echo "the Training mode was not read from modes/:"; cat "$out/training.log"; exit 1; }

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
key Down Down Down Down Return; shot 13-options # Options (after Arcade, Training, Versus, Watch)
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
key Down Down Down Down Return           # Options (after Arcade, Training, Versus, Watch)
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
key Return; key Down Down Return           # title, Versus (after Arcade, Training)
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
cp testdata/reference/chars/kfm/* "$portraits/chars/kfm/"
cp -r testdata/reference/common "$portraits/common"   # his states need common1.cns to start a match
./dist/openkakutou-linux-amd64 -content "$portraits" -width 960 -height 720 >"$out/portraits.log" 2>&1 &
pid=$!
sleep 2
key Return; key Down Down Return           # title, Versus: the select, on the fighter
shot 21-select-no-portrait
key Right; shot 22-select-portrait         # Kung Fu Man
# 1e2. Move list: Kung Fu Man (no movelist file) on both sides; the pause menu's
# Move List (4th entry) shows the commands of his .cmd over the match.
key Return; key Return; key Return         # P1, P2, stage
key Return; key Return                     # Start, skip the VS screen
sleep 2
key Escape; key Down; key Down; key Down; key Return; sleep 0.5; shot 23-movelist
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
before=$(convert "$out/21-select-no-portrait.png" -format '%[hex:u.p{342,288}]' info:)
after=$(convert "$out/22-select-portrait.png" -format '%[hex:u.p{342,288}]' info:)
[ "$before" != "$after" ] || { echo "Kung Fu Man's large portrait is not drawn: pixel 342x288 is still $after"; exit 1; }

# The move list is the screenpack's: a long list (his .cmd has more than the 14
# visible lines) shows the scrollbar, whose thumb (#FF5C38) sits at the top of
# the track at pixel 1089x200 of the 1280x800 display; text and the dimmed match
# give hundreds of colours.
thumb=$(convert "$out/23-movelist.png" -format '%[hex:u.p{1089,200}]' info:)
[ "$thumb" = "FF5C38" ] || { echo "23-movelist: the scrollbar thumb is missing: pixel is $thumb, want FF5C38"; exit 1; }
[ "$(identify -format %k "$out/23-movelist.png")" -gt 200 ] || { echo "23-movelist: too few colours for the move list"; exit 1; }

# 1f. Appearance: a second screenpack, alt, is the built-in one with its
# background image replaced by a flat #204060. Options > Appearance >
# Screenpack Right previews it (the page itself is drawn with alt, and the
# lifebar row draws a sample lifebar, P1's full bar being yellow #FAD23C),
# leaving the page applies and saves it, and a relaunch opens on its look.
alt="$out/content-alt"
cp -r testdata/content "$alt"
mkdir -p "$alt/screenpacks"
cp -r "$(go list -m -f '{{.Dir}}' github.com/openkakutou/screenpack)/default" "$alt/screenpacks/alt"
chmod -R u+w "$alt/screenpacks"
convert -size 8x8 xc:'#204060' "$alt/screenpacks/alt/images/background.png"
./dist/openkakutou-linux-amd64 -content "$alt" -width 960 -height 720 >"$out/appearance.log" 2>&1 &
pid=$!
sleep 2
key Return                       # title -> main menu
key Down Down Down Down Return           # Options (after Arcade, Training, Versus, Watch)
key Down Down Down Down Return; shot 23-appearance-default   # Appearance, drawn with the default pack
key Right; shot 24-appearance-preview-alt                    # Screenpack: alt
key Down; shot 25-appearance-lifebar-sample                  # Lifebar row: the sample
key Escape; sleep 0.5; shot 26-options-after-alt             # leave the page: applied and saved
key Escape                       # leave Options
key Down Return                  # Quit
wait $pid || { echo "the game did not quit cleanly after choosing a screenpack:"; cat "$out/appearance.log"; exit 1; }
grep -q '"screenpack": "alt"' "$alt/config/appearance.json" || { echo "appearance.json was not saved with screenpack alt:"; cat "$alt/config/appearance.json"; exit 1; }
grep -q '"lifebar": "default"' "$alt/config/appearance.json" || { echo "appearance.json lost the lifebar:"; cat "$alt/config/appearance.json"; exit 1; }
before=$(convert "$out/23-appearance-default.png" -format '%[hex:u.p{200,100}]' info:)
preview=$(convert "$out/24-appearance-preview-alt.png" -format '%[hex:u.p{200,100}]' info:)
[ "$before" = "1A1826" ] || { echo "23-appearance-default: the default pack's background is missing: pixel is $before, want 1A1826"; exit 1; }
[ "$preview" != "$before" ] || { echo "the preview did not draw the alt screenpack: pixel is still $preview"; exit 1; }
sample=$(convert "$out/25-appearance-lifebar-sample.png" -format '%[hex:u.p{310,80}]' info:)
[ "$sample" = "FAD23C" ] || { echo "the sample lifebar is not drawn: pixel is $sample, want FAD23C"; exit 1; }
applied=$(convert "$out/26-options-after-alt.png" -format '%[hex:u.p{200,100}]' info:)
[ "$applied" = "$preview" ] || { echo "the applied pack does not look like its preview: $applied vs $preview"; exit 1; }
! grep -q 'alt' "$out/appearance.log" || { echo "the alt screenpack raised a notice:"; cat "$out/appearance.log"; exit 1; }
./dist/openkakutou-linux-amd64 -content "$alt" -width 960 -height 720 >"$out/appearance2.log" 2>&1 &
pid=$!
sleep 2
shot 27-title-alt
kill -0 $pid
kill $pid; wait $pid 2>/dev/null || true
relaunch=$(convert "$out/27-title-alt.png" -format '%[hex:u.p{200,100}]' info:)
[ "$relaunch" = "$preview" ] || { echo "the relaunch did not open on the alt screenpack: pixel is $relaunch, want $preview"; exit 1; }
! grep -qi 'screenpack' "$out/appearance2.log" || { echo "the relaunch raised a screenpack notice:"; cat "$out/appearance2.log"; exit 1; }

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
for shot_name in 1-select-p1 2-select-p2 3-stage 13-options 14-options-audio 15-options-audio-music 16-options-controls 17-controls-player1 18-controls-waiting 19-controls-bound 4-setup 4b-versus 9-setup-cpu 31-arcade-versus; do
  bg=$(convert "$out/$shot_name.png" -format '%[hex:u.p{200,100}]' info:)
  [ "$bg" = "1A1826" ] || { echo "$shot_name: the screenpack background is missing: pixel is $bg, want 1A1826"; exit 1; }
  [ "$(identify -format %k "$out/$shot_name.png")" -gt 200 ] || { echo "$shot_name: too few colours for a screenpack screen"; exit 1; }
done
[ "$(identify -format %k "$out/6b-pause.png")" -gt 200 ] || { echo "6b-pause: too few colours for the screenpack pause menu"; exit 1; }

# The error screen (a capture taken 5 frames in, so still fading in) is the
# screenpack's too: its text and fading background make hundreds of colours
# where the old bitmap one had 4.
[ "$(identify -format %k "$out/8-error.png")" -gt 200 ] || { echo "8-error: too few colours for the screenpack error screen"; exit 1; }
for f in "$out"/*.png; do
  colors=$(identify -format %k "$f")
  echo "$(basename "$f"): $colors colors"
  [ "$colors" -gt 2 ] || { echo "blank screenshot: $f"; exit 1; }
done
echo "smoke OK"
