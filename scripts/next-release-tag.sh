#!/usr/bin/env bash

set -euo pipefail

release_tz="${RELEASE_TZ:-UTC}"
today_tag="$(TZ="${release_tz}" date '+v%Y.%m.%d')"

if ! git rev-parse --verify HEAD >/dev/null 2>&1; then
  echo "${today_tag}"
  exit 0
fi

head_tag="$(git tag --points-at HEAD --list 'v*' | sort -V | tail -n 1)"
if [[ -n "${head_tag}" ]]; then
  echo "${head_tag}"
  exit 0
fi

echo "${today_tag}"
