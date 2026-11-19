#!/usr/bin/env bash
# Side-by-side fidelity video: the same P1 key presses played into Ikemen GO
# (reference build) and into OpenKakutou's native game, recorded under Xvfb
# in Docker, then composed side by side.
#   run.sh --p1 path/to/A.def --p2 path/to/B.def --stage path/to/S.def --out video.mp4
#          [--work DIR] [--inputs inputs.sh] [--ik-wait 17] [--seconds 16]
#          [--ikemen-tag nightly] [--engine REF] [--character REF]
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
game="$(cd "$here/../../../.." && pwd)"           # game repo root
workspace="$(cd "$game/.." && pwd)"
work=""; inputs="$here/inputs.sh"; ik_wait=17; seconds=16; ik_tag=nightly
engine_ref=""; character_ref=""; p1=""; p2=""; stage=""; out=""
while [ $# -gt 0 ]; do
  case "$1" in
    --p1) p1="$2"; shift 2;; --p2) p2="$2"; shift 2;; --stage) stage="$2"; shift 2;;
    --out) out="$2"; shift 2;; --work) work="$2"; shift 2;; --inputs) inputs="$2"; shift 2;;
    --ik-wait) ik_wait="$2"; shift 2;; --seconds) seconds="$2"; shift 2;;
    --ikemen-tag) ik_tag="$2"; shift 2;; --engine) engine_ref="$2"; shift 2;;
    --character) character_ref="$2"; shift 2;;
    *) echo "unknown argument: $1" >&2; exit 2;;
  esac
done
[ -n "$p1" ] && [ -n "$p2" ] && [ -n "$stage" ] && [ -n "$out" ] || { sed -n 2,9p "$0" >&2; exit 2; }
for f in "$p1" "$p2" "$stage"; do [ -f "$f" ] || { echo "not a file: $f" >&2; exit 2; }; done
work="${work:-$(mktemp -d)}"; mkdir -p "$work"
engine_ref="${engine_ref:-$(git -C "$workspace/engine" rev-parse HEAD)}"
character_ref="${character_ref:-$(git -C "$workspace/character" rev-parse HEAD)}"

# 1. Recording image.
base="openkakutou-native:$(sha256sum "$game/native/Dockerfile" | cut -c1-12)"
docker image inspect "$base" >/dev/null 2>&1 || docker build -t "$base" -f "$game/native/Dockerfile" "$game/native"
image="okvideo:$(cat "$here/Dockerfile" | sha256sum | cut -c1-12)"
docker image inspect "$image" >/dev/null 2>&1 || docker build --build-arg BASE="$base" -t "$image" -f "$here/Dockerfile" "$here"
drun() { docker run --rm --user "$(id -u):$(id -g)" -e HOME=/tmp -v "$work":/w "$@"; }

# 2. Ikemen GO reference build (Linux release asset; nightly = the reference commit).
if [ ! -x "$work/ikemen/Ikemen_GO_Linux" ]; then
  gh release download "$ik_tag" -R ikemen-engine/Ikemen-GO -p 'Ikemen_GO-dev-linux.zip' -p SHA256SUMS.txt -D "$work" --clobber
  (cd "$work" && grep 'Ikemen_GO-dev-linux.zip' SHA256SUMS.txt | sha256sum -c -)
  mkdir -p "$work/ikemen" && unzip -qo "$work/Ikemen_GO-dev-linux.zip" -d "$work/ikemen"
fi
ik_commit="$(gh release view "$ik_tag" -R ikemen-engine/Ikemen-GO --json targetCommitish -q .targetCommitish)"

# 3. Content: same files for both engines. OpenKakutou wants chars/<name>/<name>.def
#    and stage files directly in stages/.
rm -rf "$work/content" "$work/ikemen/chars/okv-"* "$work/ikemen/stages/stage"
mkdir -p "$work/content/chars" "$work/content/stages" "$work/ikemen/stages/stage"
copy() {  # unreadable files (thumbnails, NAS metadata) are skipped, not fatal
  cp -r "$1" "$2" 2>/dev/null || echo "warning: some files of $1 could not be read" >&2
}
add_char() {  # $1 = .def path; prints the folder name
  local name; name="$(basename "$1" .def)"
  if [ ! -d "$work/content/chars/$name" ]; then
    copy "$(dirname "$1")" "$work/content/chars/$name"
    copy "$(dirname "$1")" "$work/ikemen/chars/okv-$name"
  fi
  echo "$name"
}
n1="$(add_char "$p1")"; n2="$(add_char "$p2")"
copy "$(dirname "$stage")/." "$work/content/stages/"
copy "$(dirname "$stage")/." "$work/ikemen/stages/stage/"
p2color=1; [ "$n1" = "$n2" ] && p2color=2
names="$(ls "$work/content/chars" | sort)"
p1idx=$(( $(echo "$names" | grep -nx "$n1" | cut -d: -f1) - 1 ))
p2idx=$(( $(echo "$names" | grep -nx "$n2" | cut -d: -f1) - 1 ))
cp "$here/inputs.sh" "$work/inputs.sh"; [ "$inputs" != "$here/inputs.sh" ] && cp "$inputs" "$work/inputs.sh"
cp "$here/record.sh" "$here/compose.sh" "$work/"

# 4. OpenKakutou native build on the requested engine/character commits.
rm -rf "$work/native"; mkdir -p "$work/native"
(cd "$game/native" && git ls-files -z | xargs -0 -I{} cp --parents {} "$work/native/")
# The Go caches live outside native/, which is replaced every run, so later runs reuse them.
mkdir -p "$work/gocache"
drun -e GOMODCACHE=/w/gocache/go-mod -e GOCACHE=/w/gocache/go-build -w /w/native "$image" bash -c \
  "go get github.com/openkakutou/engine@$engine_ref github.com/openkakutou/character@$character_ref && go build -o /w/openkakutou ./cmd/openkakutou"

# 5. Ikemen GO config: generated on first launch, then P1/P2 keys set to OpenKakutou's.
if [ ! -f "$work/ikemen/save/config.ini" ]; then
  drun -e IK_WAIT=12 -e SECONDS_REC=1 -e P1="okv-$n1" -e P1DEF="$n1.def" -e P2="okv-$n2" -e P2DEF="$n2.def" \
    -e P2COLOR=$p2color -e STAGE="$(basename "$stage")" -e P1IDX=0 -e P2IDX=0 "$image" bash /w/record.sh ikemen >/dev/null 2>&1 || true
fi
python3 "$here/ikemen_keys.py" "$work/ikemen/save/config.ini"

# 6. Record both, then compose.
env_args=(-e SECONDS_REC="$seconds" -e IK_WAIT="$ik_wait" -e P1="okv-$n1" -e P1DEF="$n1.def" -e P2="okv-$n2" -e P2DEF="$n2.def"
          -e P2COLOR=$p2color -e STAGE="$(basename "$stage")" -e P1IDX=$p1idx -e P2IDX=$p2idx)
drun "${env_args[@]}" "$image" bash /w/record.sh ikemen
drun "${env_args[@]}" "$image" bash /w/record.sh openkakutou
printf 'Ikemen GO (%s %s) - reference' "$ik_tag" "${ik_commit:0:8}" > "$work/l1.txt"
printf 'OpenKakutou (engine %s)' "${engine_ref:0:8}" > "$work/l2.txt"
printf '%s vs %s - stage %s - same P1 key presses in both engines' "$n1" "$n2" "$(basename "$stage" .def)" > "$work/l3.txt"
drun "$image" bash /w/compose.sh 2>&1 | grep -v Fontconfig || true
cp "$work/out.mp4" "$out"
cp "$work/preview.png" "${out%.*}.png"
echo "video: $out"; echo "preview: ${out%.*}.png"; echo "logs: $work/ikemen.log $work/openkakutou.log"
