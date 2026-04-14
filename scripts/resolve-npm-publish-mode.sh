#!/usr/bin/env bash

set -euo pipefail

package_name="${1:-${GIG_NPM_PACKAGE:-@phamhungptithcm/gig}}"
registry_url="${NPM_REGISTRY_URL:-https://registry.npmjs.org/}"
trusted_publishing="${NPM_TRUSTED_PUBLISHING:-}"
publish_token="${NPM_PUBLISH_TOKEN:-}"
require_mode="${GIG_NPM_REQUIRE_MODE:-false}"

package_exists=false
if npm view "${package_name}" version --registry="${registry_url}" >/dev/null 2>&1; then
  package_exists=true
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
elif [ "${trusted_publishing}" = "true" ]; then
  mode="trusted"
  reason="trusted publishing enabled"
elif [ -n "${publish_token}" ]; then
  mode="token"
  reason="NPM_PUBLISH_TOKEN fallback configured"
else
  reason="package exists but neither trusted publishing nor NPM_PUBLISH_TOKEN is configured"
fi

printf 'mode=%s\n' "${mode}"
printf 'package_exists=%s\n' "${package_exists}"
printf 'reason=%s\n' "${reason}"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    printf 'mode=%s\n' "${mode}"
    printf 'package_exists=%s\n' "${package_exists}"
    printf 'reason=%s\n' "${reason}"
  } >> "${GITHUB_OUTPUT}"
fi

if [ "${require_mode}" = "true" ] && [ "${mode}" = "none" ]; then
  echo "npm publish is not configured for ${package_name}." >&2
  echo "Reason: ${reason}" >&2
  echo "Configure one of these release paths:" >&2
  echo "  1. Set repository secret NPM_PUBLISH_TOKEN for the first publish or token fallback." >&2
  echo "  2. After the package exists, configure npm trusted publishing and set NPM_TRUSTED_PUBLISHING=true." >&2
  exit 1
fi
