#!/usr/bin/env bash

set -euo pipefail

package_name="${1:-}"
release_tag="${2:-}"
registry_url="${NPM_REGISTRY_URL:-https://registry.npmjs.org/}"
max_attempts="${NPM_VERIFY_MAX_ATTEMPTS:-18}"
sleep_seconds="${NPM_VERIFY_SLEEP_SECONDS:-10}"

if [ -z "${package_name}" ] || [ -z "${release_tag}" ]; then
  echo "usage: verify-published-npm-version.sh <package-name> <release-tag>" >&2
  exit 1
fi

if [[ ! "${release_tag}" =~ ^v([0-9]{4})\.([0-9]{2})\.([0-9]{2})$ ]]; then
  echo "release tag must use vYYYY.MM.DD: ${release_tag}" >&2
  exit 1
fi

year="${BASH_REMATCH[1]}"
month="${BASH_REMATCH[2]}"
day="${BASH_REMATCH[3]}"
expected_version="${year}.$((10#${month})).$((10#${day}))"
package_ref="${package_name}@${expected_version}"

attempt=1
last_error=""
while [ "${attempt}" -le "${max_attempts}" ]; do
  err_file="$(mktemp)"
  if published_version="$(npm view "${package_ref}" version --registry="${registry_url}" 2>"${err_file}")"; then
    rm -f "${err_file}"
    published_version="$(printf '%s\n' "${published_version}" | tail -n 1 | tr -d '\r')"
    if [ "${published_version}" = "${expected_version}" ]; then
      echo "Published ${package_name}@${published_version}"
      exit 0
    fi

    echo "Published npm version ${published_version} did not match expected ${expected_version}." >&2
    exit 1
  fi

  last_error="$(cat "${err_file}")"
  rm -f "${err_file}"

  if printf '%s' "${last_error}" | grep -qiE 'E404|Not found|could not be found'; then
    if [ "${attempt}" -lt "${max_attempts}" ]; then
      echo "Package ${package_ref} is not visible on npm yet. Retrying (${attempt}/${max_attempts})..." >&2
      sleep "${sleep_seconds}"
      attempt=$((attempt + 1))
      continue
    fi
  fi

  printf '%s\n' "${last_error}" >&2
  exit 1
done

echo "Package ${package_name}@${expected_version} was not visible on npm after ${max_attempts} attempts." >&2
if [ -n "${last_error}" ]; then
  printf '%s\n' "${last_error}" >&2
fi
exit 1
