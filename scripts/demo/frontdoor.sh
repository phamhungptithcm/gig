#!/usr/bin/env bash

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_ROOT="$(mktemp -d)"
BIN_DIR="$TMP_ROOT/bin"
FAKE_BIN_DIR="$TMP_ROOT/fake-bin"
FIXTURE_ROOT="$TMP_ROOT/demo"
DEER_FLOW_FIXTURE="$FIXTURE_ROOT/deer-flow"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

mkdir -p "$BIN_DIR" "$FAKE_BIN_DIR" "$DEER_FLOW_FIXTURE/backend" "$DEER_FLOW_FIXTURE/frontend"

for command in make docker uv pnpm python3; do
  cat >"$FAKE_BIN_DIR/$command" <<'EOF'
#!/bin/sh
exit 0
EOF
  chmod +x "$FAKE_BIN_DIR/$command"
done

cp "$ROOT/deer-flow/Makefile" "$DEER_FLOW_FIXTURE/Makefile"
cp "$ROOT/deer-flow/config.example.yaml" "$DEER_FLOW_FIXTURE/config.example.yaml"
cp "$ROOT/deer-flow/frontend/.env.example" "$DEER_FLOW_FIXTURE/frontend/.env.example"

(
  cd "$ROOT"
  go build -o "$BIN_DIR/gig" ./cmd/gig
)

export PATH="$FAKE_BIN_DIR:$PATH"
export PATH="$BIN_DIR:$PATH"
export GIG_WORKAREA_FILE="$TMP_ROOT/workareas.json"

run() {
  printf '\n$ %s\n' "$*"
  "$@"
}

run gig
run gig workarea add payments --repo github:acme/payments --from staging --to main --use
run gig
run gig assist doctor --path "$FIXTURE_ROOT"
run gig assist setup --path "$FIXTURE_ROOT"
run gig assist doctor --path "$FIXTURE_ROOT"
