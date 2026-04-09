#!/usr/bin/env bash

set -euo pipefail

previous_tag="${1:-}"
current_tag="${2:?usage: release-notes.sh [previous-tag] <current-tag>}"

is_date_tag() {
  [[ "${1:-}" =~ ^v[0-9]{4}\.[0-9]{2}\.[0-9]{2}$ ]]
}

print_initial_release_notes() {
  cat <<'EOF'
## Summary

`gig` helps teams answer one release question before moving code forward:

**Did we miss any change for this ticket?**

This release strengthens `gig` as a ticket-aware release reconciliation CLI for teams working across multiple repositories and repeated QA/UAT feedback loops.

## Why it matters

In real delivery workflows, one ticket rarely ends with one clean commit in one repository.

A single ticket may touch backend, frontend, database, scripts, or low-code assets across multiple repos. It may also fail QA or client review more than once, creating late follow-up fixes that are easy to miss during promotion.

That is where release risk usually appears:
- one repository is forgotten
- one late fix is not included
- a DB or config change is not reviewed
- release notes and checklists are assembled manually from scattered sources

`gig` reduces that reconciliation work by giving teams a consistent, read-only workflow before promotion.

## What teams can do with `gig`

- scan a workspace for repositories
- inspect one ticket across repositories
- compare source and target branches for a ticket
- verify release readiness with `safe`, `warning`, or `blocked`
- build a read-only promotion plan
- generate Markdown or JSON release packets
- validate team config and repo mapping with `gig doctor`

## Value for teams

For developers, `gig` reduces the time spent checking whether all follow-up commits for a ticket are actually accounted for across repositories.

For release engineers and QA/UAT coordinators, `gig` reduces manual branch-by-branch reconciliation and makes it easier to see what is missing, what is risky, and what still needs manual review.

For delivery leads, `gig` improves confidence and auditability by turning ticket-based release checks into a repeatable workflow instead of an ad hoc process.

## What changes without `gig`

Without `gig`, teams typically have to inspect repositories one by one, search ticket commits manually, compare branches by hand, and assemble release communication themselves. That process is repetitive, slow, and vulnerable to missed details.

With `gig`, teams can inspect, verify, plan, and document ticket promotion in a single CLI workflow that is safer by default and easier to operationalize.

## Notes

`gig` is intentionally read-only today. It does not cherry-pick, merge, or deploy for you. The current focus is to make release decisions clearer and safer before any write action happens.
EOF
}

normalize_commit_subject() {
  local subject="${1}"

  subject="${subject#feat: }"
  subject="${subject#fix: }"
  subject="$(printf '%s' "${subject}" | sed -E 's/^(feat|fix|refactor|perf|docs|chore|build|ci|test)(\([^)]*\))?!?:[[:space:]]*//')"
  printf '%s\n' "${subject}"
}

commit_range="HEAD"
if [[ -n "${previous_tag}" ]]; then
  if ! git rev-parse -q --verify "refs/tags/${previous_tag}" >/dev/null 2>&1; then
    echo "unknown previous tag: ${previous_tag}" >&2
    exit 1
  fi
  commit_range="${previous_tag}..HEAD"
fi

if [[ -z "${previous_tag}" ]] || ! is_date_tag "${previous_tag}"; then
  print_initial_release_notes
  exit 0
fi

feature_changes=()
bugfix_changes=()

while IFS= read -r subject; do
  [[ -n "${subject}" ]] || continue

  normalized_subject="$(normalize_commit_subject "${subject}")"
  if printf '%s\n' "${subject}" | grep -Eq '^fix(\([^)]*\))?!?:[[:space:]]*'; then
    bugfix_changes+=("${normalized_subject}")
    continue
  fi

  feature_changes+=("${normalized_subject}")
done < <(git log --no-merges --reverse --format='%s' "${commit_range}")

cat <<EOF
## Summary

This release continues to improve \`gig\` as a ticket-aware release reconciliation CLI for multi-repository delivery workflows.

## Feature
EOF

if [[ ${#feature_changes[@]} -eq 0 ]]; then
  echo "- No feature updates captured in this release."
else
  for change in "${feature_changes[@]}"; do
    echo "- ${change}"
  done
fi

echo
echo "## Bug Fix"

if [[ ${#bugfix_changes[@]} -eq 0 ]]; then
  echo "- No bug fixes captured in this release."
else
  for change in "${bugfix_changes[@]}"; do
    echo "- ${change}"
  done
fi
