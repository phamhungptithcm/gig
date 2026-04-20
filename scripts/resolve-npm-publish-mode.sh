#!/usr/bin/env bash

set -euo pipefail

package_name="${1:-${GIG_NPM_PACKAGE:-@hunpeolabs/gig}}"
registry_url="${NPM_REGISTRY_URL:-https://registry.npmjs.org/}"
publish_token="${NPM_PUBLISH_TOKEN:-}"
require_mode="${GIG_NPM_REQUIRE_MODE:-false}"
github_repo="${GIG_GITHUB_REPO:-phamhungptithcm/gig}"
workflow_file="${GIG_NPM_WORKFLOW_FILE:-release.yml}"
environment_name="${GIG_NPM_ENVIRONMENT:-npm-release}"

package_exists=false
if npm view "${package_name}" version --registry="${registry_url}" >/dev/null 2>&1; then
  package_exists=true
fi

trusted_publisher_configured=false
if [ "${package_exists}" = "true" ]; then
  trust_err="$(mktemp)"
  if trust_json="$(
    npm trust list "${package_name}" --json --registry="${registry_url}" 2>"${trust_err}"
  )"; then
    if printf '%s' "${trust_json}" \
      | node "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/npm-publish-state.cjs" \
        trust-match "${github_repo}" "${workflow_file}" "${environment_name}"; then
      trusted_publisher_configured=true
    fi
  fi
  rm -f "${trust_err}"
fi

mode="none"
reason="npm publish is not configured"

if [ "${package_exists}" = "false" ]; then
  if [ -n "${publish_token}" ]; then
    mode="token"
    reason="bootstrap first publish with NPM_PUBLISH_TOKEN"
  else
    reason="package does not exist on npm yet; set NPM_PUBLISH_TOKEN for the first publish"
  fi
elif [ "${trusted_publisher_configured}" = "true" ]; then
  mode="trusted"
  reason="trusted publishing is configured for phamhungptithcm/gig release.yml on npm-release"
elif [ -n "${publish_token}" ]; then
  mode="token"
  reason="trusted publishing is not configured for this package; using NPM_PUBLISH_TOKEN fallback"
else
  reason="package exists but no matching trusted publisher or NPM_PUBLISH_TOKEN fallback is configured"
fi

printf 'mode=%s\n' "${mode}"
printf 'package_exists=%s\n' "${package_exists}"
printf 'trusted_publisher_configured=%s\n' "${trusted_publisher_configured}"
printf 'reason=%s\n' "${reason}"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    printf 'mode=%s\n' "${mode}"
    printf 'package_exists=%s\n' "${package_exists}"
    printf 'trusted_publisher_configured=%s\n' "${trusted_publisher_configured}"
    printf 'reason=%s\n' "${reason}"
  } >> "${GITHUB_OUTPUT}"
fi

if [ "${require_mode}" = "true" ] && [ "${mode}" = "none" ]; then
  echo "npm publish is not configured for ${package_name}." >&2
  echo "Reason: ${reason}" >&2
  echo "Configure one of these release paths:" >&2
  echo "  1. Set repository secret NPM_PUBLISH_TOKEN for the first publish or token fallback." >&2
  echo "     The token must be an npm automation token or a granular token with bypass 2FA enabled." >&2
  echo "  2. After the package exists, configure npm trusted publishing for phamhungptithcm/gig / release.yml / npm-release." >&2
  exit 1
fi
