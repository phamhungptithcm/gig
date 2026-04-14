#!/usr/bin/env bash

set -euo pipefail

package_name="${1:-${GIG_NPM_PACKAGE:-@phamhungptithcm/gig}}"
registry_url="${NPM_REGISTRY_URL:-https://registry.npmjs.org/}"

if [ -z "${NODE_AUTH_TOKEN:-}" ]; then
  echo "NODE_AUTH_TOKEN is required for npm publish auth verification." >&2
  exit 1
fi

npm_user="$(npm whoami --registry="${registry_url}")"
printf 'npm_user=%s\n' "${npm_user}"

package_exists=false
if npm view "${package_name}" version --registry="${registry_url}" >/dev/null 2>&1; then
  package_exists=true
fi
printf 'package_exists=%s\n' "${package_exists}"

case "${package_name}" in
  @*/*)
    scope_name="${package_name#@}"
    scope_name="${scope_name%%/*}"
    scope_ref="@${scope_name}"
    printf 'scope=%s\n' "${scope_ref}"

    if [ "${package_exists}" = "false" ]; then
      access_err="$(mktemp)"
      if ! npm access list packages "${scope_ref}" --json --registry="${registry_url}" >/dev/null 2>"${access_err}"; then
        echo "Unable to verify npm scope access for ${scope_ref}." >&2
        cat "${access_err}" >&2
        rm -f "${access_err}"
        echo "The first publish requires an npm user or org that owns ${scope_ref}." >&2
        echo "Use a token from that owner, or rename the package to a scope you control." >&2
        exit 1
      fi
      rm -f "${access_err}"
    fi
    ;;
esac
