"""Count the common effects and sounds that real characters reference.

Usage: python3 tools/common-assets/census.py CORPUS_DIR... > tools/common-assets/CENSUS.md

Scans the state files (.cns .st .cmd .zss) of every directory given, reads
only numbers and writes counts: no third-party content is copied. A character
is the nearest ancestor directory holding a .def file.
"""
import collections
import os
import re
import sys

STATE_EXTENSIONS = (".cns", ".st", ".cmd", ".zss")
KEY = re.compile(r"^\s*(sparkno|guard\.sparkno|hitsound|guardsound|value|anim)\s*[=:]\s*([^;\n]*)", re.I)
TYPE = re.compile(r"^\s*type\s*[=:]\s*(\w+)", re.I)
SECTION = re.compile(r"^\s*\[")
PAIR = re.compile(r"^\s*([sf]?)\s*(-?\d+)\s*,\s*(-?\d+)\s*$", re.I)
ANIM = re.compile(r"^\s*([sf]?)\s*(-?\d+)\s*$", re.I)
OWN = re.compile(r"^\s*s\s*\d", re.I)  # S-prefixed expression such as S8200+(random%8)

SPARK_TYPES = ("hitdef", "reversaldef")
ANIM_TYPES = ("explod", "changeanim", "changeanim2", "helper")

spark_counts = collections.Counter()   # fightfx action -> references
spark_chars = collections.defaultdict(set)
sound_counts = collections.Counter()   # common (group, sample) -> references
sound_chars = collections.defaultdict(set)
anim_counts = collections.Counter()    # fightfx action from F-prefixed anims
anim_chars = collections.defaultdict(set)
totals = collections.Counter()
roots = {}


def character_of(path):
    folder = os.path.dirname(path)
    chain = []
    while folder not in roots:
        chain.append(folder)
        try:
            has_def = any(name.lower().endswith(".def") for name in os.listdir(folder))
        except OSError:
            has_def = False
        if has_def:
            roots[folder] = folder
            break
        parent = os.path.dirname(folder)
        if parent == folder:
            roots[folder] = chain[0]
            break
        folder = parent
    found = roots[folder]
    for item in chain:
        roots[item] = found
    return found


def note(counter, chars, key, char):
    counter[key] += 1
    chars[key].add(char)


def handle_block(kind, entries, char):
    if kind in SPARK_TYPES:
        totals["hitdef blocks"] += 1
        if not any(k in ("sparkno", "guard.sparkno") for k, _ in entries):
            totals["hitdef blocks without sparkno (engine default)"] += 1
    for key, raw in entries:
        text = raw.split(";")[0].strip()
        if key in ("sparkno", "guard.sparkno") and kind in SPARK_TYPES:
            m = ANIM.match(text)
            if m is None and OWN.match(text):
                totals["spark from the character (S prefix)"] += 1
            elif m is None:
                totals["dynamic spark value (expression)"] += 1
            elif m.group(1).lower() == "s":
                totals["spark from the character (S prefix)"] += 1
            else:
                note(spark_counts, spark_chars, int(m.group(2)), char)
        elif key in ("hitsound", "guardsound") and kind in SPARK_TYPES:
            m = PAIR.match(text)
            if m is None and OWN.match(text):
                totals["hit/guard sound from the character (S prefix)"] += 1
            elif m is None:
                totals["dynamic hit/guard sound value (expression)"] += 1
            elif m.group(1).lower() == "s":
                totals["hit/guard sound from the character (S prefix)"] += 1
            else:
                note(sound_counts, sound_chars, (int(m.group(2)), int(m.group(3))), char)
        elif key == "value" and kind == "playsnd":
            m = PAIR.match(text)
            if m is not None and m.group(1).lower() == "f":
                note(sound_counts, sound_chars, (int(m.group(2)), int(m.group(3))), char)
            elif m is None:
                totals["dynamic PlaySnd value (expression)"] += 1
        elif key in ("anim", "value") and kind in ANIM_TYPES:
            m = ANIM.match(text)
            if m is not None and m.group(1).lower() == "f":
                note(anim_counts, anim_chars, int(m.group(2)), char)


def scan(path):
    char = character_of(path)
    kind = ""
    entries = []
    with open(path, "r", encoding="latin-1", errors="replace") as handle:
        for line in handle:
            if SECTION.match(line):
                handle_block(kind, entries, char)
                kind, entries = "", []
                continue
            t = TYPE.match(line)
            if t:
                kind = t.group(1).lower()
                continue
            k = KEY.match(line)
            if k:
                entries.append((k.group(1).lower(), k.group(2)))
    handle_block(kind, entries, char)
    totals["state files"] += 1


def main(dirs):
    chars_seen = set()
    for top in dirs:
        for folder, _, names in os.walk(top, onerror=lambda e: totals.update({"unreadable folders": 1})):
            for name in names:
                if name.lower().endswith(STATE_EXTENSIONS):
                    try:
                        scan(os.path.join(folder, name))
                        chars_seen.add(character_of(os.path.join(folder, name)))
                    except OSError:
                        totals["unreadable files"] += 1
    totals["characters with state files"] = len(chars_seen)
    report()


def table(title, counter, chars, fmt):
    print(f"\n## {title}\n")
    print("| number | references | characters |")
    print("|---|---|---|")
    for key in sorted(counter):
        print(f"| {fmt(key)} | {counter[key]} | {len(chars[key])} |")


def report():
    print("# Census of common effects and sounds referenced by characters\n")
    print("Generated by `tools/common-assets/census.py` over the Mugen character corpus and the")
    print("reference characters under `engine/testdata/reference/chars` and")
    print("`native/testdata/reference/chars`. Numbers and counts only.\n")
    print("## Totals\n")
    for name in sorted(totals):
        print(f"- {name}: {totals[name]}")
    table("Fightfx actions used as spark numbers (HitDef, ReversalDef: sparkno, guard.sparkno, no S prefix)",
          spark_counts, spark_chars, str)
    table("Common sounds (group,sample): hitsound/guardsound without S prefix, PlaySnd with F prefix",
          sound_counts, sound_chars, lambda k: f"{k[0]},{k[1]}")
    table("Fightfx actions used as F-prefixed anims (Explod, ChangeAnim, ChangeAnim2, Helper)",
          anim_counts, anim_chars, str)


main(sys.argv[1:])
