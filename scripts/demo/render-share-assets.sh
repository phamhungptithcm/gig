#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
GIF_PATH="${1:-$ROOT/docs/assets/gig-showcase.gif}"
MP4_PATH="${2:-$ROOT/docs/assets/gig-showcase.mp4}"
TMP_DIR="$(mktemp -d)"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if ! command -v qlmanage >/dev/null 2>&1; then
  echo "qlmanage not found."
  echo "This script currently expects macOS Quick Look for SVG rendering."
  exit 1
fi

if ! command -v ffmpeg >/dev/null 2>&1; then
  echo "ffmpeg not found."
  echo "Install ffmpeg to build the showcase MP4 and GIF."
  exit 1
fi

mkdir -p "$(dirname "$GIF_PATH")" "$(dirname "$MP4_PATH")"
mkdir -p "$TMP_DIR/renders"

render_svg() {
  local svg_path="$1"
  qlmanage -t -s 1200 -o "$TMP_DIR/renders" "$svg_path" >/dev/null 2>&1
}

render_svg "$ROOT/docs/assets/front-door.svg"
render_svg "$ROOT/docs/assets/ticket-inspect-demo.svg"
render_svg "$ROOT/docs/assets/ticket-verify-demo.svg"
render_svg "$ROOT/docs/assets/ticket-assist-demo.svg"

ffmpeg -y \
  -loop 1 -t 2.8 -i "$TMP_DIR/renders/front-door.svg.png" \
  -loop 1 -t 2.8 -i "$TMP_DIR/renders/ticket-inspect-demo.svg.png" \
  -loop 1 -t 2.8 -i "$TMP_DIR/renders/ticket-verify-demo.svg.png" \
  -loop 1 -t 2.8 -i "$TMP_DIR/renders/ticket-assist-demo.svg.png" \
  -filter_complex "[0:v]scale=960:-2,setsar=1[v0];[1:v]scale=960:-2,setsar=1[v1];[2:v]scale=960:-2,setsar=1[v2];[3:v]scale=960:-2,setsar=1[v3];[v0][v1]xfade=transition=fade:duration=0.4:offset=2.4[x1];[x1][v2]xfade=transition=fade:duration=0.4:offset=4.8[x2];[x2][v3]xfade=transition=fade:duration=0.4:offset=7.2,format=yuv420p[v]" \
  -map "[v]" \
  -r 12 \
  -movflags +faststart \
  "$MP4_PATH" >/dev/null 2>&1

ffmpeg -y \
  -i "$MP4_PATH" \
  -vf "fps=10,scale=800:-1:flags=lanczos,palettegen" \
  "$TMP_DIR/palette.png" >/dev/null 2>&1

ffmpeg -y \
  -i "$MP4_PATH" \
  -i "$TMP_DIR/palette.png" \
  -filter_complex "fps=10,scale=800:-1:flags=lanczos[x];[x][1:v]paletteuse=dither=bayer:bayer_scale=3" \
  "$GIF_PATH" >/dev/null 2>&1

echo "Saved showcase MP4 to $MP4_PATH"
echo "Saved showcase GIF to $GIF_PATH"
