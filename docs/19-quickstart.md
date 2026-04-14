# Quick Start

This page is for the first few minutes with `gig`.

If you only remember one thing, remember this path:

`install -> login -> inspect -> verify -> export`

## 1. Install

Use the direct installer until the first npm bootstrap publish is live:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
gig version
```

If `@hunpeolabs/gig` is already available in your environment, this also works:

```bash
npm install -g @hunpeolabs/gig
gig version
```

If npm returns `404`, the first package publish has not completed yet.

## 2. Open The Front Door

```bash
gig
```

If you are in an interactive terminal, `gig` opens a picker-first front door:

- use `↑/↓` to move
- press `Enter` or `Space` to select
- start with GitHub if you are not sure what to pick
- use the current folder when you already have a repo checked out locally
- or type directly into the command palette: `ABC-123`, `inspect ABC-123`, `verify ABC-123`, `repo github:owner/name ABC-123`

The goal is that a new user can get to the first audit without learning repo target syntax first.

## 3. Log In Once

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

Use the provider that matches the repository you want to audit.

## 4. Inspect One Ticket

```bash
gig inspect ABC-123 --repo github:owner/name
```

Supported remote target forms:

- `github:owner/name`
- `gitlab:group/project`
- `bitbucket:workspace/repo`
- `azure-devops:org/project/repo`
- `svn:https://svn.example.com/repos/app/branches/staging/ProductName`

Use `inspect` when you want the full ticket picture in one place:

- repositories touched
- commits found
- branches containing those commits
- risk hints such as DB or config changes

## 5. Verify The Next Move

```bash
gig verify --ticket ABC-123 --repo github:owner/name
```

Use `verify` when you want a release decision instead of raw evidence:

- `safe`
- `warning`
- `blocked`

`gig` will try to infer the likely promotion path before you reach for `--from` or `--to`.

## 6. Export A Release Packet

```bash
gig manifest generate --ticket ABC-123 --repo github:owner/name
```

Use this when you want release-ready output for QA, release review, or downstream tooling without rewriting the audit by hand.

## 7. Optional: Save A Workarea

```bash
gig workarea add payments --repo github:owner/name --from staging --to main --use
gig inspect ABC-123
gig verify --ticket ABC-123
```

Use a workarea when you want `gig` to remember project scope and defaults so repeated commands stay short.
You do not need this on day one because `gig` can also remember successful remote usage automatically.

## 8. Optional: Add An AI Briefing Layer

If you want an audience-specific explanation on top of the deterministic audit bundle:

```bash
gig assist doctor
gig assist setup
gig assist audit --ticket ABC-123 --repo github:owner/name --audience release-manager
```

The AI layer is additive.
`gig` stays the source of truth.

## 9. Local Fallback When Needed

Use local mode when remote access is not enough yet:

```bash
gig scan --path .
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --path .
gig manifest generate --ticket ABC-123 --path .
```

## 10. Only Add Config If Inference Needs Help

Most teams should not start with `gig.yaml`.

Add config only when you need:

- branch topology overrides
- repo metadata such as service names or owners
- team notes that should appear in output

## Demo

For a stable terminal walkthrough that is good for README updates, portfolio clips, or documentation:

```bash
./scripts/demo/frontdoor.sh
./scripts/demo/record-frontdoor.sh
```

See [Demo Guide](25-demo-guide.md) and [Portfolio Guide](26-portfolio-guide.md) for publishing advice.
