# Quick Start

This page is for the first 5 minutes.

If you only want to know what `gig` does and how to try it fast, start here.

## What `gig` Helps You Answer

Before release, teams often ask:

- did we miss any change for this ticket?
- is `test` behind `dev` for this ticket?
- is `main` missing a follow-up fix?
- does this ticket include DB or config work that needs manual review?

`gig` helps answer those questions from your Git history in a simple, read-only way.

## First Command To Run

```bash
gig --help
```

That shows the full command list and the main usage patterns.

If you already know your ticket ID, these are the two most useful commands:

```bash
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
```

## What Each Command Is For

- `gig scan`
  Find repositories under a folder.
- `gig find`
  Find commits for one ticket.
- `gig inspect`
  Show the full ticket picture across repositories.
- `gig env status`
  Show where the ticket is present or behind across environment branches.
- `gig diff`
  Compare one branch to another for a single ticket.
- `gig verify`
  Tell you if a promotion looks `safe`, `warning`, or `blocked`.
- `gig plan`
  Build a read-only promotion plan for people or CI tools.
- `gig version`
  Show the installed version.

## A Good First Workflow

### 1. Scan your workspace

```bash
gig scan --path .
```

### 2. Inspect the ticket

```bash
gig inspect ABC-123 --path .
```

### 3. Check environment status

```bash
gig env status ABC-123 --path . --envs dev=dev,test=test,prod=main
```

### 4. Verify whether the next move is safe

```bash
gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
```

### 5. Generate a plan or JSON manifest

```bash
gig plan --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
gig plan --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main --format json
```

## When To Use `inspect`, `verify`, And `plan`

- use `inspect` when you want to know what changed
- use `verify` when you want a quick go or no-go signal
- use `plan` when you want the next release step written out clearly

## What `gig` Does Not Do Yet

Right now, `gig` does not move code for you.

It does not cherry-pick, merge, or deploy.

That is intentional.
The current product focus is to help teams make safer release decisions before any write action happens.

## Where To Go Next

- read [CLI Guide](03-cli-spec.md) for full command help
- read [Branching And Release](15-branching-and-release.md) to understand this repo's own workflow
- read [Roadmap](13-roadmap.md) to see what comes next
