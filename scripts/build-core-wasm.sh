#!/usr/bin/env bash
# Build the game core (native/cmd/core-wasm) to public/wasm/core.wasm and copy
# the matching Go runtime glue to public/wasm/core-wasm_exec.js. Uses the Go
# on PATH when there is one (CI: actions/setup-go, go-version-file
# native/go.mod), else the native toolchain container.
set -euo pipefail
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
out="$root/public/wasm"
mkdir -p "$out"
if command -v go >/dev/null 2>&1; then
  (cd "$root/native" && GOOS=js GOARCH=wasm go build -buildvcs=false -o "$out/core.wasm" ./cmd/core-wasm)
  cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" "$out/core-wasm_exec.js"
else
  tmp="$root/native/.cache/core-wasm"
  rm -rf "$tmp" && mkdir -p "$tmp"
  "$root/native/scripts/docker.sh" bash -c 'GOOS=js GOARCH=wasm go build -buildvcs=false -o .cache/core-wasm/core.wasm ./cmd/core-wasm && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .cache/core-wasm/core-wasm_exec.js'
  cp "$tmp/core.wasm" "$tmp/core-wasm_exec.js" "$out/"
  rm -rf "$tmp"
fi
