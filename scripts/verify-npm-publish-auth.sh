#!/usr/bin/env bash

set -euo pipefail

package_name="${1:-${GIG_NPM_PACKAGE:-@hunpeolabs/gig}}"
registry_url="${NPM_REGISTRY_URL:-https://registry.npmjs.org/}"
helper_script="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/npm-publish-state.cjs"

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

if [ "${package_exists}" = "true" ]; then
  collab_err="$(mktemp)"
  if ! collaborators_json="$(
    npm access list collaborators "${package_name}" "${npm_user}" --json --registry="${registry_url}" 2>"${collab_err}"
  )"; then
    echo "Unable to verify publish access for ${package_name} as npm user ${npm_user}." >&2
    cat "${collab_err}" >&2
    rm -f "${collab_err}"
    echo "For an existing package, prefer npm trusted publishing or use a token from a package owner/collaborator with write access." >&2
    exit 1
  fi
  rm -f "${collab_err}"

  if ! access_level="$(
    printf '%s' "${collaborators_json}" | node "${helper_script}" collaborator-access "${npm_user}"
  )"; then
    echo "npm user ${npm_user} does not have read-write publish access to ${package_name}." >&2
    echo "For an existing package, prefer npm trusted publishing or replace NPM_PUBLISH_TOKEN with a package owner or collaborator token that has write access." >&2
    exit 1
  fi

  if [ "${access_level}" != "read-write" ]; then
    echo "npm user ${npm_user} only has ${access_level} access to ${package_name}; publish requires read-write access." >&2
    echo "For an existing package, prefer npm trusted publishing or replace NPM_PUBLISH_TOKEN with a package owner or collaborator token that has write access." >&2
    exit 1
  fi

  printf 'package_access=%s\n' "${access_level}"
fi

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
