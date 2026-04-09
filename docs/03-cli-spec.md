# CLI Guide

This page explains what each command does and when to use it.

If you are new, start with `gig --help` and then come back here.

## Start Here

### Show all commands

```bash
gig --help
```

### Get help for one command

```bash
gig inspect --help
gig verify --help
gig plan --help
```

You can also use:

```bash
gig help
```

## Commands At A Glance

| Command | Use it when you want to... |
| --- | --- |
| `gig scan` | find repos under a folder |
| `gig find` | find commits for one ticket |
| `gig inspect` | see the full ticket story across repos |
| `gig env status` | check where a ticket is present or behind across env branches |
| `gig diff` | compare one branch to another for a ticket |
| `gig verify` | get a quick `safe`, `warning`, or `blocked` result |
| `gig plan` | build a promotion plan for people or CI |
| `gig version` | confirm what build you installed |

## Shared Rules

- commands print human-readable output by default
- errors go to stderr
- successful commands exit with code `0`
- usage errors return code `2`
- runtime failures return code `1`
- current commands are read-only and never change repositories

## `gig scan`

Use this when you first want to know what repos `gig` can see.

```bash
gig scan --path .
```

What it shows:

- repo name
- SCM type
- current branch when known
- repo path

## `gig find`

Use this when you only want the raw commit list for one ticket.

```bash
gig find ABC-123 --path .
```

What it shows:

- matching commits
- commit messages
- branches containing those commits when available

## `gig inspect`

Use this when you want the full ticket picture across repos.

```bash
gig inspect ABC-123 --path .
```

What it shows:

- repos touched by the ticket
- ticket commit count per repo
- branches where those commits appear
- risk signals such as DB, config, or Mendix-style changes

## `gig env status`

Use this when you want to see where a ticket stands across the environment line.

```bash
gig env status ABC-123 --path . --envs dev=dev,test=test,prod=main
```

What it shows:

- whether the ticket is present in each environment branch
- whether the next environment is behind
- which repos still need attention before promotion

## `gig diff`

Use this when you want a simple branch-to-branch comparison for one ticket.

```bash
gig diff --ticket ABC-123 --from dev --to test --path .
```

What it shows:

- commits found in the source branch
- commits already present in the target branch
- commits still missing in the target branch

## `gig verify`

Use this when you want a fast release decision.

```bash
gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
```

Optional JSON output:

```bash
gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main --format json
```

What it shows:

- one overall verdict: `safe`, `warning`, or `blocked`
- why the tool gave that verdict
- per-repo checks
- manual-review steps when risky files are detected

## `gig plan`

Use this when you want a clear, read-only promotion plan.

```bash
gig plan --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
```

Optional JSON output:

```bash
gig plan --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main --format json
```

What it shows:

- per-repo verdict
- commits expected to move next
- environment status in the selected flow
- risk signals
- manual steps
- planned actions

This JSON output is the first release-manifest style output in the project.

## `gig version`

Use this when you want to confirm what build is installed.

```bash
gig version
```

## Common Examples

### See what changed for a ticket

```bash
gig inspect ABC-123 --path /path/to/workspace
```

### Check whether `test` is behind `dev`

```bash
gig env status ABC-123 --path /path/to/workspace --envs dev=dev,test=test,prod=main
```

### Check whether it is safe to move from `test` to `main`

```bash
gig verify --ticket ABC-123 --from test --to main --path /path/to/workspace --envs dev=dev,test=test,prod=main
```

### Generate a JSON plan for CI or review tooling

```bash
gig plan --ticket ABC-123 --from test --to main --path /path/to/workspace --envs dev=dev,test=test,prod=main --format json
```

## Output Formats

- `human`
  easy to read in terminal and good for manual review
- `json`
  good for CI, scripts, and future release packet tooling

JSON output is currently available on:

- `gig verify`
- `gig plan`

## Exit Codes

- `0`: success
- `1`: runtime failure
- `2`: usage error
- `3`: reserved for future partial-success execution flows

## What Is Planned Next

Planned next CLI additions include:

- `gig manifest generate`
- `gig doctor`
- `gig plan --release <release-id>`
