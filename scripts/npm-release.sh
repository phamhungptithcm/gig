#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
package_name="${GIG_NPM_PACKAGE:-@phamhungptithcm/gig}"
github_repo="${GIG_GITHUB_REPO:-phamhungptithcm/gig}"
workflow_file="${GIG_NPM_WORKFLOW_FILE:-release.yml}"
environment_name="${GIG_NPM_ENVIRONMENT:-npm-release}"

usage() {
  cat <<EOF
Usage:
  ./scripts/npm-release.sh prepare [release-tag]
  ./scripts/npm-release.sh publish-first [release-tag]
  ./scripts/npm-release.sh trust
  ./scripts/npm-release.sh bootstrap [release-tag]
  ./scripts/npm-release.sh verify

Examples:
  ./scripts/npm-release.sh prepare v2026.04.13
  ./scripts/npm-release.sh prepare
  ./scripts/npm-release.sh bootstrap v2026.04.13
  ./scripts/npm-release.sh bootstrap
  ./scripts/npm-release.sh verify
EOF
}

require_command() {
  local name="${1}"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "${name} is required" >&2
    exit 1
  fi
}

require_npm_login() {
  if ! npm whoami >/dev/null 2>&1; then
    echo "npm login is required. Run: npm login" >&2
    exit 1
  fi
}

require_npm_trust_support() {
  local version
  version="$(npm --version)"
  if ! node -e '
const current = process.argv[1].split(".").map(Number);
const required = [11, 10, 0];
for (let i = 0; i < required.length; i += 1) {
  const left = current[i] || 0;
  const right = required[i];
  if (left > right) process.exit(0);
  if (left < right) process.exit(1);
}
process.exit(0);
' "${version}"; then
    echo "npm@11.10.0 or newer is required for trusted publishing. Current: ${version}" >&2
    exit 1
  fi
}

load_release_tags() {
  local tags=""

  if command -v gh >/dev/null 2>&1; then
    tags="$(gh release list -R "${github_repo}" --limit 200 2>/dev/null | awk '{print $1}' | grep '^v' || true)"
  fi

  if [ -z "${tags}" ] && command -v git >/dev/null 2>&1; then
    tags="$(git -C "${repo_root}" tag --list 'v*' --sort=-v:refname || true)"
  fi

  printf '%s\n' "${tags}" | sed '/^$/d'
}

select_release_tag_with_fzf() {
  local tags="${1}"
  local selected=""

  selected="$(
    printf '%s\n' "${tags}" | fzf \
      --prompt="Release tag> " \
      --height=40% \
      --layout=reverse \
      --border \
      --preview-window=right:60%:wrap \
      --preview '
        repo="'"${github_repo}"'"
        tag="{}"
        if command -v gh >/dev/null 2>&1; then
          gh release view "$tag" -R "$repo" --json tagName,name,publishedAt,isDraft,isPrerelease \
            --template "Tag: {{.tagName}}\nName: {{.name}}\nPublished: {{.publishedAt}}\nDraft: {{.isDraft}}\nPrerelease: {{.isPrerelease}}\n" 2>/dev/null || true
        fi
      '
  )"

  printf '%s\n' "${selected}"
}

select_release_tag_manually() {
  local tags="${1}"
  local query=""
  local filtered=""
  local selection=""
  local count=0

  echo "fzf is not installed, using search + numbered selection."
  printf "Search release tag (blank for all): "
  IFS= read -r query

  if [ -n "${query}" ]; then
    filtered="$(printf '%s\n' "${tags}" | grep -iF "${query}" || true)"
  else
    filtered="${tags}"
  fi

  if [ -z "${filtered}" ]; then
    echo "No release tags matched your search." >&2
    return 1
  fi

  echo
  while IFS= read -r tag; do
    count=$((count + 1))
    printf "%2d. %s\n" "${count}" "${tag}"
  done <<< "${filtered}"
  echo

  printf "Choose a release tag by number: "
  IFS= read -r selection
  if ! [[ "${selection}" =~ ^[0-9]+$ ]]; then
    echo "Selection must be a number." >&2
    return 1
  fi

  printf '%s\n' "${filtered}" | sed -n "${selection}p"
}

resolve_release_tag() {
  local provided="${1:-}"
  local tags=""
  local selected=""

  if [ -n "${provided}" ]; then
    printf '%s\n' "${provided}"
    return
  fi

  tags="$(load_release_tags)"
  if [ -z "${tags}" ]; then
    echo "No release tags were found. Pass a tag explicitly, for example: v2026.04.13" >&2
    exit 1
  fi

  if [ -t 0 ] && [ -t 1 ]; then
    if command -v fzf >/dev/null 2>&1; then
      selected="$(select_release_tag_with_fzf "${tags}")"
    else
      selected="$(select_release_tag_manually "${tags}")"
    fi
  else
    selected="$(printf '%s\n' "${tags}" | head -n 1)"
  fi

  if [ -z "${selected}" ]; then
    echo "No release tag selected." >&2
    exit 1
  fi

  printf '%s\n' "${selected}"
}

prepare_package() {
  local release_tag="${1}"
  node "${repo_root}/scripts/prepare-npm-package.cjs" "${release_tag}" "${repo_root}/dist/npm-package"
  npm pack --dry-run "${repo_root}/dist/npm-package"
}

publish_first() {
  local release_tag="${1}"
  require_npm_login
  prepare_package "${release_tag}"
  npm publish "${repo_root}/dist/npm-package" --access public
}

configure_trust() {
  require_npm_login
  require_npm_trust_support
  npm trust github "${package_name}" --repo "${github_repo}" --file "${workflow_file}" --env "${environment_name}" --yes
}

print_next_steps() {
  cat <<EOF

Next steps:
1. Set GitHub repository variable:
   gh variable set NPM_TRUSTED_PUBLISHING --repo ${github_repo} --body true
2. Run the Release workflow on main.
3. After CI has run on main and staging, sync required checks:
   ./scripts/sync-required-checks.sh ${github_repo}
EOF
}

verify_setup() {
  if npm view "${package_name}" name version 2>/dev/null; then
    echo
    echo "Trusted publishers:"
    npm trust list "${package_name}" || true
    return
  fi

  cat <<EOF
Package ${package_name} is not published yet.

Run the first bootstrap publish:
  ./scripts/npm-release.sh bootstrap
EOF
}

main() {
  require_command node
  require_command npm

  if [ "$#" -lt 1 ]; then
    usage >&2
    exit 1
  fi

  case "${1}" in
    prepare)
      [ "$#" -le 2 ] || {
        echo "prepare accepts at most one optional [release-tag]" >&2
        exit 1
      }
      prepare_package "$(resolve_release_tag "${2:-}")"
      ;;
    publish-first)
      [ "$#" -le 2 ] || {
        echo "publish-first accepts at most one optional [release-tag]" >&2
        exit 1
      }
      publish_first "$(resolve_release_tag "${2:-}")"
      ;;
    trust)
      [ "$#" -eq 1 ] || {
        echo "trust does not accept extra arguments" >&2
        exit 1
      }
      configure_trust
      ;;
    bootstrap)
      [ "$#" -le 2 ] || {
        echo "bootstrap accepts at most one optional [release-tag]" >&2
        exit 1
      }
      publish_first "$(resolve_release_tag "${2:-}")"
      configure_trust
      verify_setup
      print_next_steps
      ;;
    verify)
      [ "$#" -eq 1 ] || {
        echo "verify does not accept extra arguments" >&2
        exit 1
      }
      verify_setup
      ;;
    -h|--help|help)
      usage
      ;;
    *)
      echo "unknown command: ${1}" >&2
      usage >&2
      exit 1
      ;;
  esac
}

main "$@"
