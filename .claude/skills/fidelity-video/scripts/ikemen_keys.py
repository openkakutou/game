"""Set Ikemen GO's P1/P2 keyboard keys to OpenKakutou's native defaults, so
one xdotool script drives both engines (game/native/internal/gfx/input.go)."""
import re
import sys

KEYS = {
    "Keys_P1": dict(up="w", down="s", left="a", right="d", a="f", b="g", c="h", x="r", y="t", z="y", start="RETURN"),
    "Keys_P2": dict(up="UP", down="DOWN", left="LEFT", right="RIGHT", a="j", b="k", c="l", x="i", y="o", z="p", start="RSHIFT"),
}

path = sys.argv[1]
text = open(path, encoding="utf-8").read()
for section, keys in KEYS.items():
    body = "".join(f"{k:<8} = {v}\n" for k, v in keys.items())
    body += "d        = Not used\nw        = Not used\nmenu     = Not used\n"
    text = re.sub(rf"\[{section}\]\nJoystick = -1\nGUID     = \n.*?\n\n",
                  f"[{section}]\nJoystick = -1\nGUID     = \n{body}\n", text, flags=re.S)
open(path, "w", encoding="utf-8").write(text)
