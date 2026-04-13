#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT_PATH="${1:-$ROOT/docs/assets/gig-demo.cast}"

if ! command -v asciinema >/dev/null 2>&1; then
  echo "asciinema not found; generating a deterministic cast instead."
  python3 "$ROOT/scripts/demo/build_cast.py" "$OUTPUT_PATH" "$ROOT/scripts/demo/frontdoor.sh"
  echo "Saved cast to $OUTPUT_PATH"
  exit 0
fi

mkdir -p "$(dirname "$OUTPUT_PATH")"

asciinema rec \
  --overwrite \
  --idle-time-limit 1.5 \
  --title "gig terminal demo" \
  -c "$ROOT/scripts/demo/frontdoor.sh" \
  "$OUTPUT_PATH"

echo "Saved cast to $OUTPUT_PATH"
