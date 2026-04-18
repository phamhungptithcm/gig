#!/usr/bin/env bash
set -euo pipefail

runs=5
ticket_id="ABC-123"
gig_bin=""
keep_workspace="false"

usage() {
  cat <<'EOF'
Usage: scripts/benchmark-release-audit.sh [--runs N] [--ticket ABC-123] [--gig /path/to/gig] [--keep-workspace]

Creates a synthetic multi-repo release workspace, then compares:
- a manual git-based audit loop
- one `gig verify` command

The output shows average elapsed time, command count, and the step reduction.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --runs)
      runs="${2:?missing value for --runs}"
      shift 2
      ;;
    --ticket)
      ticket_id="${2:?missing value for --ticket}"
      shift 2
      ;;
    --gig)
      gig_bin="${2:?missing value for --gig}"
      shift 2
      ;;
    --keep-workspace)
      keep_workspace="true"
      shift
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

now_ms() {
  python3 - <<'PY'
import time
print(int(time.time() * 1000))
PY
}

avg_ms() {
  python3 - "$@" <<'PY'
import sys
values = [int(value) for value in sys.argv[1:]]
print(sum(values) // len(values))
PY
}

git_commit() {
  local repo_root="$1"
  local message="$2"
  git -C "$repo_root" add -A >/dev/null
  git -C "$repo_root" commit -m "$message" >/dev/null
}

create_repo() {
  local repo_root="$1"
  local file_path="$2"
  local initial_body="$3"
  local ticket_body="$4"
  local follow_up_body="$5"

  mkdir -p "$repo_root"
  git init -q -b develop "$repo_root"
  git -C "$repo_root" config user.name "gig benchmark"
  git -C "$repo_root" config user.email "gig-benchmark@example.com"

  mkdir -p "$(dirname "$repo_root/$file_path")"
  printf '%s\n' "$initial_body" >"$repo_root/$file_path"
  git_commit "$repo_root" "chore: initial scaffold"

  git -C "$repo_root" branch staging
  git -C "$repo_root" branch main

  git -C "$repo_root" checkout -q staging
  printf '%s\n' "$ticket_body" >>"$repo_root/$file_path"
  git_commit "$repo_root" "$ticket_id add staged release work"

  printf '%s\n' "$follow_up_body" >>"$repo_root/$file_path"
  git_commit "$repo_root" "$ticket_id follow-up fix after QA"

  git -C "$repo_root" checkout -q develop
}

workspace_dir="$(mktemp -d)"
if [[ "$keep_workspace" != "true" ]]; then
  trap 'rm -rf "$workspace_dir"' EXIT
fi

create_repo "$workspace_dir/payments-api" "service/app.txt" \
  "api bootstrap" "payment validation change" "payment validation follow-up"
create_repo "$workspace_dir/payments-ui" "ui/app.txt" \
  "ui bootstrap" "checkout summary update" "checkout copy follow-up"

if [[ -z "$gig_bin" ]]; then
  gig_bin="$workspace_dir/gig"
  go build -o "$gig_bin" ./cmd/gig
fi

manual_audit() {
  local repo_root="$1"
  local from_branch="$2"
  local to_branch="$3"
  git -C "$repo_root" log "$from_branch" --grep "$ticket_id" --pretty='%H' >/dev/null
  git -C "$repo_root" log "$to_branch" --grep "$ticket_id" --pretty='%H' >/dev/null
  git -C "$repo_root" cherry -v "$to_branch" "$from_branch" | grep "$ticket_id" >/dev/null || true
}

run_manual_benchmark() {
  manual_audit "$workspace_dir/payments-api" staging main
  manual_audit "$workspace_dir/payments-ui" staging main
}

run_gig_benchmark() {
  "$gig_bin" verify --ticket "$ticket_id" --path "$workspace_dir" --from staging --to main >/dev/null
}

manual_timings=()
gig_timings=()

for ((run = 1; run <= runs; run++)); do
  start_ms="$(now_ms)"
  run_manual_benchmark
  end_ms="$(now_ms)"
  manual_timings+=("$((end_ms - start_ms))")

  start_ms="$(now_ms)"
  run_gig_benchmark
  end_ms="$(now_ms)"
  gig_timings+=("$((end_ms - start_ms))")
done

manual_avg_ms="$(avg_ms "${manual_timings[@]}")"
gig_avg_ms="$(avg_ms "${gig_timings[@]}")"

manual_commands=6
gig_commands=1
manual_steps=7
gig_steps=1
command_reduction="$(python3 - <<PY
manual_commands = $manual_commands
gig_commands = $gig_commands
print(f"{manual_commands / gig_commands:.1f}x")
PY
)"
elapsed_ratio="$(python3 - <<PY
manual_ms = max($manual_avg_ms, 1)
gig_ms = max($gig_avg_ms, 1)
print(f"{manual_ms / gig_ms:.1f}x")
PY
)"
elapsed_note="$(python3 - <<PY
manual_ms = $manual_avg_ms
gig_ms = $gig_avg_ms
ratio = max(manual_ms, 1) / max(gig_ms, 1)
if manual_ms > gig_ms:
    print(f"gig was faster in this synthetic run ({ratio:.1f}x manual/gig ratio)")
elif gig_ms > manual_ms:
    print(f"the manual loop was faster on this tiny local synthetic run ({1/ratio:.1f}x gig/manual ratio)")
else:
    print("manual and gig were effectively tied in this synthetic run")
PY
)"

cat <<EOF
gig release-audit benchmark

workspace: $workspace_dir
ticket:    $ticket_id
runs:      $runs
gig:       $gig_bin

Scenario              Avg ms   Commands   Notes
manual git loop       $manual_avg_ms      $manual_commands         grep + cherry in each repo
gig verify            $gig_avg_ms      $gig_commands         one release verdict command

Step reduction:       $command_reduction fewer commands with gig
Human steps:          $manual_steps manual steps vs $gig_steps gig step
Elapsed ratio:        $elapsed_ratio manual/gig
Elapsed note:         $elapsed_note

Notes:
- This benchmark is synthetic and measures repeatable command work, not human context-switch cost.
- The manual path is intentionally representative: repo-by-repo search plus branch comparison.
- For release-day team usage, the real gap is usually larger because humans also read and reconcile the output.
- On very small local workspaces, raw milliseconds may favor the manual loop even when gig removes most of the human steps.
EOF
