#!/usr/bin/env bash

set -euo pipefail

release_tz="${RELEASE_TZ:-UTC}"
calendar_month="$(TZ="${release_tz}" date '+%Y.%m')"
IFS='.' read -r release_year release_month <<< "${calendar_month}"
month_tag="v${release_year}.$((10#${release_month}))"
legacy_month_tag="v${release_year}.${release_month}"

if ! git rev-parse --verify HEAD >/dev/null 2>&1; then
  echo "${month_tag}.0"
  exit 0
fi

head_tag="$(git tag --points-at HEAD --list 'v*' | sort -V | tail -n 1)"
if [[ -n "${head_tag}" ]]; then
  echo "${head_tag}"
  exit 0
fi

month_regex="${month_tag//./\\.}"
legacy_month_regex="${legacy_month_tag//./\\.}"
highest_micro=-1

while IFS= read -r tag; do
  if [[ "${tag}" =~ ^${month_regex}\.([0-9]+)$ ]]; then
    micro=$((10#${BASH_REMATCH[1]}))
    if (( micro > highest_micro )); then
      highest_micro="${micro}"
    fi
  fi
done < <(git tag --list "${month_tag}.*")

if [[ "${legacy_month_tag}" != "${month_tag}" ]]; then
  while IFS= read -r tag; do
    if [[ "${tag}" =~ ^${legacy_month_regex}\.([0-9]+)$ ]]; then
      micro=$((10#${BASH_REMATCH[1]}))
      if (( micro > highest_micro )); then
        highest_micro="${micro}"
      fi
    fi
  done < <(git tag --list "${legacy_month_tag}.*")
fi

printf '%s.%d\n' "${month_tag}" "$((highest_micro + 1))"
