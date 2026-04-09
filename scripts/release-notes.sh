#!/usr/bin/env bash

set -euo pipefail

previous_tag="${1:-}"
current_tag="${2:?usage: release-notes.sh [previous-tag] <current-tag>}"
release_date="$(date -u +%Y-%m-%d)"

echo "# ${current_tag}"
echo
echo "Release date: ${release_date}"
echo

if [[ -n "${previous_tag}" ]]; then
  echo "Changes since ${previous_tag}."
else
  echo "Initial release."
fi

echo
echo "## Included Changes"
echo

if [[ -n "${previous_tag}" ]]; then
  commits="$(git log --no-merges --reverse --format='- %s (%h)' "${previous_tag}..HEAD")"
else
  commits="$(git log --no-merges --reverse --format='- %s (%h)')"
fi

if [[ -n "${commits}" ]]; then
  echo "${commits}"
else
  echo "- No code changes captured for this release."
fi

echo
echo "## Validation"
echo
echo "- go test ./..."
echo "- go build -o bin/gig ./cmd/gig"
