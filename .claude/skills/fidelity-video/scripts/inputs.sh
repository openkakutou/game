# Default P1 input script, played identically into both engines with xdotool.
# P1 keys (same in both): W A S D move, F G H = a b c, R T Y = x y z.
# Written for a P1 standing on the left, facing right.
k() { xdotool keydown "$@"; sleep 0.10; xdotool keyup "$@"; }
hold() { xdotool keydown "$1"; sleep "$2"; xdotool keyup "$1"; }
sleep 0.5
hold d 1.0                 # walk forward
sleep 0.3; k f; sleep 0.6  # a
k g; sleep 0.8             # b
k r; sleep 0.8             # x
xdotool keydown s; sleep 0.2; k h; xdotool keyup s; sleep 0.8   # crouching c
hold a 0.8                 # walk back
k w; sleep 1.2             # jump
xdotool keydown s; sleep 0.05; xdotool keydown d; sleep 0.05; xdotool keyup s; sleep 0.05; k f; xdotool keyup d; sleep 1.5  # D, DF, F + a
xdotool keydown d; sleep 0.05; xdotool keyup d; xdotool keydown s; sleep 0.05; xdotool keydown d; sleep 0.05; xdotool keyup s; k r; xdotool keyup d; sleep 1.5  # F, D, DF + x
sleep 1
