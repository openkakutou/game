# Runs inside the recording container. $1 = ikemen | openkakutou.
# Env: P1, P2 (character folder names), P2COLOR, STAGE (stage .def name),
# IK_WAIT (seconds until Ikemen GO's "Fight!"), SECONDS_REC, P1IDX, P2IDX.
set -u
Xvfb :99 -screen 0 1280x720x24 >/dev/null 2>&1 &
export DISPLAY=:99 LIBGL_ALWAYS_SOFTWARE=1 SDL_AUDIODRIVER=dummy
sleep 1
if [ "$1" = ikemen ]; then
  cd /w/ikemen
  LD_LIBRARY_PATH=/w/ikemen/lib ./Ikemen_GO_Linux -p1 "chars/$P1/$P1DEF" -p2 "chars/$P2/$P2DEF" -p2.color "$P2COLOR" \
    -s "stages/stage/$STAGE" -rounds 1 > /w/ikemen.log 2>&1 &
  sleep "$IK_WAIT"
else
  cd /w
  ./openkakutou -content /w/content -width 1280 -height 720 > /w/openkakutou.log 2>&1 &
  sleep 3
  for _ in $(seq 1 "$P1IDX"); do xdotool key Right; sleep 0.2; done; xdotool key Return; sleep 0.5
  for _ in $(seq 1 "$P2IDX"); do xdotool key Right; sleep 0.2; done; xdotool key Return; sleep 0.5
  xdotool key Return; sleep 0.5; xdotool key Return   # stage, then Start
  sleep 1
fi
ffmpeg -loglevel error -y -f x11grab -framerate 60 -video_size 1280x720 -i :99 -t "$SECONDS_REC" \
  -c:v libx264 -preset veryfast -pix_fmt yuv420p "/w/$1.mp4" &
rec=$!
bash /w/inputs.sh
wait $rec
