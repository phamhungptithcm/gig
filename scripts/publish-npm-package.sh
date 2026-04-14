#!/usr/bin/env bash

set -euo pipefail

package_dir="${1:-}"

if [ -z "${package_dir}" ]; then
  echo "usage: publish-npm-package.sh <package-dir>" >&2
  exit 1
fi

package_dir="$(cd "${package_dir}" && pwd)"
stderr_log="$(mktemp)"
pack_dir="$(mktemp -d)"
cleanup() {
  rm -f "${stderr_log}"
  rm -rf "${pack_dir}"
}
trap cleanup EXIT

tarball_name="$(
  cd "${pack_dir}"
  npm pack "${package_dir}" | tail -n 1
)"
tarball_path="${pack_dir}/${tarball_name}"

if [ ! -f "${tarball_path}" ]; then
  echo "Failed to create npm tarball from ${package_dir}." >&2
  exit 1
fi

set +e
npm publish "${tarball_path}" --access public 2> >(tee "${stderr_log}" >&2)
status=$?
set -e

if [ "${status}" -eq 0 ]; then
  exit 0
fi

if grep -qi 'bypass 2fa' "${stderr_log}"; then
  echo "NPM_PUBLISH_TOKEN must be an npm automation token or a granular token with bypass 2FA enabled." >&2
fi

if grep -qi 'Scope not found' "${stderr_log}"; then
  echo "The authenticated npm account does not control the target package scope." >&2
  echo "Use a token from the owner of the scope, or publish under a scope that already exists." >&2
fi

exit "${status}"
