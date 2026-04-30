# Local Fallback

Local mode uses a checked-out Git or SVN repository.
It is useful, but it should not be the first mental model when remote provider access is available.

## Inspect Locally

```bash
gig ABC-123 --path .
```

Use this when you are already inside a checkout and only need ticket evidence.

## Verify Locally

```bash
gig verify ABC-123 --path . --from staging --to main
```

Local `verify` needs a source and target branch unless a project or config supplies them.

## Generate A Local Manifest

```bash
gig packet ABC-123 --path . --from staging --to main
```

## Save Local Defaults

```bash
gig project add local-payments --path . --from staging --to main --use
gig verify ABC-123
gig packet ABC-123
```

## SVN And Mendix

Local SVN remains useful for teams with SVN/Mendix layouts.

```bash
gig ABC-123 --path .
gig verify ABC-123 --path . --from release/2026.04 --to trunk
```

If your SVN branch layout is nested, prefer explicit branch names that match the team release path.

## When Local Mode Is The Right Choice

- remote provider access is blocked
- the repo is already checked out
- you need local conflict state
- SVN/Mendix path details are easier to inspect locally
- you are debugging branch topology before saving a project

## When Local Mode Is Not Enough

Local mode may not include provider-only evidence such as:

- PR or merge-request metadata
- hosted deployment status
- linked issues or work items
- release metadata

Use remote mode when that evidence matters.
