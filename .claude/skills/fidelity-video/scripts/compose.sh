# Runs inside the recording container: both recordings side by side, labelled.
set -e
cd /w
F=/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf
ffmpeg -loglevel error -y -i ikemen.mp4 -i openkakutou.mp4 -filter_complex \
"[0:v]scale=960:540,pad=960:600:0:60:color=0x111111,drawtext=fontfile=$F:textfile=l1.txt:fontcolor=white:fontsize=26:x=(w-tw)/2:y=18[a];\
[1:v]scale=960:540,pad=960:600:0:60:color=0x111111,drawtext=fontfile=$F:textfile=l2.txt:fontcolor=white:fontsize=26:x=(w-tw)/2:y=18[b];\
[a][b]hstack,pad=1920:660:0:0:color=0x111111,drawtext=fontfile=$F:textfile=l3.txt:fontcolor=0xbbbbbb:fontsize=22:x=(w-tw)/2:y=624" \
  -c:v libx264 -preset medium -crf 23 -pix_fmt yuv420p -movflags +faststart out.mp4
ffmpeg -loglevel error -y -ss "${PREVIEW_AT:-7}" -i out.mp4 -frames:v 1 -vf scale=1280:-1 preview.png
