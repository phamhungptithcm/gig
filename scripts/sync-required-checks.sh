#!/usr/bin/env bash

set -euo pipefail

repo="${1:-phamhungptithcm/gig}"
shift || true

if [ "$#" -eq 0 ]; then
  branches=(main staging)
else
  branches=("$@")
fi

required_checks=("go" "packaging" "docs")

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required" >&2
  exit 1
fi

for branch in "${branches[@]}"; do
  echo "Checking required CI runs on ${repo}@${branch}..."
  payload="$(gh api "repos/${repo}/commits/${branch}/check-runs")"

  PAYLOAD="${payload}" python3 - "$branch" "${required_checks[@]}" <<'PY'
import json
import os
import sys

branch = sys.argv[1]
required = sys.argv[2:]
payload = json.loads(os.environ["PAYLOAD"])
runs = payload.get("check_runs", [])
latest = {}
for run in runs:
    name = run.get("name", "")
    if name not in latest or run.get("started_at", "") > latest[name].get("started_at", ""):
        latest[name] = run

missing = []
failing = []
for name in required:
    run = latest.get(name)
    if not run:
        missing.append(name)
        continue
    conclusion = run.get("conclusion")
    status = run.get("status")
    if status != "completed" or conclusion not in {"success", "neutral", "skipped"}:
        failing.append(f"{name}={status}/{conclusion}")

if missing or failing:
    if missing:
        print(f"{branch}: missing checks: {', '.join(missing)}", file=sys.stderr)
    if failing:
        print(f"{branch}: non-passing checks: {', '.join(failing)}", file=sys.stderr)
    sys.exit(1)
PY

  args=("api" "-X" "PATCH" "repos/${repo}/branches/${branch}/protection/required_status_checks" "-f" "strict=true")
  for check in "${required_checks[@]}"; do
    args+=("-f" "contexts[]=${check}")
  done

  echo "Updating required checks on ${branch}: ${required_checks[*]}"
  gh "${args[@]}" >/dev/null
done

echo "Required status checks are now synced for: ${branches[*]}"
