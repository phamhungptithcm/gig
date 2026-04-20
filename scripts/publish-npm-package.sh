#!/usr/bin/env bash

set -euo pipefail

package_dir="${1:-}"

if [ -z "${package_dir}" ]; then
  echo "usage: publish-npm-package.sh <package-dir>" >&2
  exit 1
fi

package_dir="$(cd "${package_dir}" && pwd)"
package_name="$(
  node -p "require(process.argv[1]).name || ''" "${package_dir}/package.json"
)"
registry_url="${NPM_REGISTRY_URL:-https://registry.npmjs.org/}"
stdout_log="$(mktemp)"
stderr_log="$(mktemp)"
pack_dir="$(mktemp -d)"
cleanup() {
  rm -f "${stdout_log}"
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
npm publish "${tarball_path}" --access public --registry "${registry_url}" >"${stdout_log}" 2>"${stderr_log}"
status=$?
set -e

if [ -s "${stdout_log}" ]; then
  cat "${stdout_log}"
fi

if [ -s "${stderr_log}" ]; then
  cat "${stderr_log}" >&2
fi

if [ "${status}" -eq 0 ]; then
  exit 0
fi

if grep -qi 'bypass 2fa' "${stderr_log}"; then
  echo "NPM_PUBLISH_TOKEN must be an npm automation token or a granular token with bypass 2FA enabled." >&2
fi

if grep -qiE 'E404|Not Found - PUT|do not have permission to access it' "${stderr_log}"; then
  package_exists=false
  if [ -n "${package_name}" ] && npm view "${package_name}" version --registry "${registry_url}" >/dev/null 2>&1; then
    package_exists=true
  fi

  if [ "${package_exists}" = "true" ]; then
    echo "${package_name} already exists on npm, but the authenticated identity cannot publish it." >&2
    echo "Prefer npm trusted publishing for steady-state releases, or replace NPM_PUBLISH_TOKEN with a package owner or collaborator token that has write access." >&2
  fi
fi

if grep -qi 'Scope not found' "${stderr_log}"; then
  echo "The authenticated npm account does not control the target package scope." >&2
  echo "Use a token from the owner of the scope, or publish under a scope that already exists." >&2
fi

exit "${status}"
