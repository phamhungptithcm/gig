#!/usr/bin/env bash

set -euo pipefail

if ! git rev-parse --verify HEAD >/dev/null 2>&1; then
  echo "v0.1.0"
  exit 0
fi

head_tag="$(git tag --points-at HEAD --list 'v*' | sort -V | tail -n 1)"
if [[ -n "${head_tag}" ]]; then
  echo "${head_tag}"
  exit 0
fi

latest_tag="$(git tag --list 'v*' --sort=-v:refname | head -n 1)"
if [[ -z "${latest_tag}" ]]; then
  echo "v0.1.0"
  exit 0
fi

version="${latest_tag#v}"
IFS=. read -r major minor patch <<<"${version}"
patch="$((patch + 1))"

echo "v${major}.${minor}.${patch}"
