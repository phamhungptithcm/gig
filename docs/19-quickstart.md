# Quick Start

This page is for the first 5 minutes.

If you want to know what `gig` does and try it fast, start here.

## What `gig` Helps You Answer

Before release, teams often ask:

- did we miss any change for this ticket?
- is `test` behind `dev` for this ticket?
- is `main` missing a late follow-up fix?
- does this ticket include DB, config, or Mendix work that needs manual review?

`gig` helps answer those questions from repository history in a simple, read-only way.

Today that means:

- Git for the full flow
- SVN for `scan`, `find`, and `inspect`

## First Command To Run

```bash
gig --help
```

That shows the full command list and the main usage patterns.

## The 5 Commands Most Teams Need First

```bash
gig scan --path .
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path .
gig manifest generate --ticket ABC-123 --from test --to main --path .
gig doctor --path .
```

## A Good First Workflow

### 1. Scan your workspace

```bash
gig scan --path .
```

Use this to see what repos `gig` can detect.

### 2. Inspect one ticket

```bash
gig inspect ABC-123 --path .
```

Use this to see which repos changed, what commits were found, and whether risky files were touched.

### 3. Check whether the next move looks safe

```bash
gig verify --ticket ABC-123 --from test --to main --path .
```

Use this when you want a quick release decision:

- `safe`
- `warning`
- `blocked`

### 4. Generate a release packet people can actually read

```bash
gig manifest generate --ticket ABC-123 --from test --to main --path .
```

This produces a Markdown packet with:

- a short summary
- QA checklist
- client review notes
- release manager checklist
- per-repo details, risks, notes, and commits to include

### 5. Check whether your config is good enough to trust

```bash
gig doctor --path .
```

This checks things like:

- is a config file present?
- do repo catalog entries match real repos?
- do configured environment branches exist?
- are service, owner, and kind filled in?

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

It also does not read Jira or deployment evidence yet.

That is intentional.
The current product focus is to help teams make safer release decisions first.

## Where To Go Next

- read [CLI Guide](03-cli-spec.md) for full command help
- read [Config Spec](09-config-spec.md) to map your real workflow
- read [Roadmap](13-roadmap.md) to see what is next
