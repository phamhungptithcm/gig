# Quick Start

This page is for the first 5 minutes.

If you want to know what `gig` does and try it fast, start here.

The product direction is moving toward:

`install -> login -> choose workarea -> inspect`

The commands below stay within what the current build can already do today.

## Install

```bash
npm install -g @phamhungptithcm/gig
gig version
```

If npm returns `404`, the package has not completed its first bootstrap publish yet.
Use the direct installer from the README or trigger the next `main` release after npm publishing is configured.

## What `gig` Helps You Answer

Before release, teams often ask:

- did we miss any change for this ticket?
- is `test` behind `dev` for this ticket?
- is `main` missing a late follow-up fix?
- does this ticket include DB, config, or Mendix work that needs manual review?

`gig` helps answer those questions from repository history in a simple, read-only way.

Today that means:

- GitHub, GitLab, Bitbucket, and Azure DevOps-backed remote inspection for supported live flows
- local Git and SVN workspaces for the broader read-only flow
- local Git conflict inspection and optional AI conflict briefings when Git is already stopped on a conflict

## First Command To Run

```bash
gig
```

That opens the guided front door with the current workarea, next commands, and the optional DeerFlow doctor plus setup hints.

## Full Command List

```bash
gig --help
```

That shows the full command list and the compact help patterns.

## The Fastest Current Flow

```bash
gig login github
gig workarea add --provider github --use
gig inspect ABC-123
gig verify --ticket ABC-123
gig assist doctor
gig assist setup
gig manifest generate --ticket ABC-123
gig assist audit --ticket ABC-123 --audience release-manager
```

Supported remote repository targets today:

- `github:owner/name`
- `gitlab:group/project`
- `bitbucket:workspace/repo`
- `azure-devops:org/project/repo`
- `svn:https://svn.example.com/repos/app/branches/staging/ProductName`

## A Good First Workflow

### 1. Connect to a provider once

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

Use this when you want live inspection without cloning first.

### 2. Save a workarea for one project

```bash
gig workarea add --provider github --use
gig workarea add payments --repo github:owner/name --from staging --to main --use
gig workarea list
gig workarea use payments
```

Use this when you want `gig` to remember repo scope and promotion defaults so later commands can stay short.
If you omit `--repo` and `--path`, `gig` can discover a repository from a logged-in GitHub, GitLab, Bitbucket, or Azure DevOps account and let you choose it interactively.
The picker accepts either a number or filter text, and recent workareas or repositories are promoted to the top.

### 3. Inspect one ticket directly on a remote repository or the current workarea

```bash
gig inspect ABC-123
gig inspect ABC-123 --repo github:owner/name
gig inspect ABC-123 --repo gitlab:group/project
gig inspect ABC-123 --repo bitbucket:workspace/repo
gig inspect ABC-123 --repo azure-devops:org/project/repo
gig inspect ABC-123 --repo svn:https://svn.example.com/repos/app/branches/staging/ProductName
```

Use this to see what changed, which branches contain the ticket, and whether risky files were touched.
On GitHub, GitLab, Bitbucket, and Azure DevOps, `gig` also shows pull request and deployment evidence when the provider can confirm it.

### 4. Check whether the next move looks safe

```bash
gig verify --ticket ABC-123
gig verify --ticket ABC-123 --repo github:owner/name
```

Use this when you want a quick release decision:

- `safe`
- `warning`
- `blocked`

### 5. Generate a release packet people can actually read

```bash
gig manifest generate --ticket ABC-123
gig manifest generate --ticket ABC-123 --repo github:owner/name
```

This produces a Markdown packet with:

- a short summary
- QA checklist
- client review notes
- release manager checklist
- per-repo details, risks, notes, and commits to include

### 5.1 Optional: Ask DeerFlow for an audience-specific ticket briefing

If the bundled DeerFlow sidecar is not configured yet, bootstrap it first:

```bash
gig assist doctor
gig assist setup
```

```bash
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
gig assist audit --ticket ABC-123 --repo github:owner/name --audience client
gig assist audit --ticket ABC-123 --repo github:owner/name --audience release-manager
```

Use this experimental command when you want the same ticket evidence explained for QA, a client, or a release manager without changing the deterministic core.

### 5.2 Optional: Ask DeerFlow for a release-level briefing

```bash
gig assist release --release rel-2026-04-09 --path . --audience release-manager
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

Use the first form after you save ticket snapshots into the same release.
Use the second form when you want `gig` to build the release bundle live from a ticket file and a remote repository target.

### 5.3 Optional: Ask DeerFlow for an active conflict brief

```bash
gig assist resolve --path . --ticket ABC-123 --audience release-manager
```

Use this when Git has already stopped on a merge, rebase, or cherry-pick conflict and you want `gig` to explain the active block and its risks before you choose an action.

### 5.4 Optional: Run the deterministic terminal demo

```bash
./scripts/demo/frontdoor.sh
./scripts/demo/record-frontdoor.sh
```

Use this when you want a stable screencast or README demo without relying on live provider APIs.

### 6. If needed, fall back to local workspace mode

```bash
gig scan --path .
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path .
gig manifest generate --ticket ABC-123 --from test --to main --path .
gig doctor --path .
gig resolve status --path .
gig assist resolve --path . --ticket ABC-123 --audience release-manager
gig resolve start --path .
```

Use local mode when:

- remote provider access is not enough yet
- you need broad workspace scanning
- your team depends on explicit branch overrides or repo catalog metadata

## If Your Team Uses Real Branch Names

You do not have to keep passing `--envs` manually.

Create a `gig.yaml` like this:

```yaml
ticketPattern: '\b[A-Z][A-Z0-9]+-\d+\b'

environments:
  - name: dev
    branch: develop
  - name: test
    branch: release/test
  - name: prod
    branch: main

repositories:
  - path: services/accounts-api
    service: Accounts API
    owner: Backend Team
    kind: app
    notes:
      - Verify login and billing summary in QA.
```

Then run:

```bash
gig verify --ticket ABC-123 --from test --to main --path .
gig manifest generate --ticket ABC-123 --from test --to main --path .
gig doctor --path .
```

There is a sample file in the repo:

- [gig.example.yaml](https://github.com/phamhungptithcm/gig/blob/main/gig.example.yaml)

## What `gig` Does Not Do Yet

Right now, `gig` does not move code for you.

It does not cherry-pick, merge, or deploy.

It now includes an initial workarea flow, but it does not yet deliver the full guided project picker and richer console navigation the product is aiming for.

That is intentional.
The current product focus is to help teams make safer release decisions first.

## Where To Go Next

- read [CLI Guide](03-cli-spec.md) for full command help
- read [Agent Skills](24-agent-skills.md) for project-specific agent workflows
- read [Config Spec](09-config-spec.md) to map your real workflow
- read [Roadmap](13-roadmap.md) to see what is next
