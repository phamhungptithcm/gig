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
gig manifest --help
gig doctor --help
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
| `gig plan` | build a read-only promotion plan for people or CI |
| `gig manifest generate` | generate a Markdown or JSON release packet |
| `gig doctor` | check config coverage, env mapping, and repo catalog health |
| `gig version` | confirm what build you installed |

## Shared Rules

- commands print human-readable output by default unless you ask for JSON
- errors go to stderr
- successful commands exit with code `0`
- usage errors return code `2`
- runtime failures return code `1`
- current commands are read-only and never change repositories
- `--ticket-file` accepts one ticket ID per line and ignores blank lines plus lines that start with `#`

## Shared Config Behavior

Ticket-aware commands can load config automatically.

Supported file names:

- `gig.yaml`
- `gig.yml`
- `.gig.yaml`
- `.gig.yml`

`gig` searches upward from the path you pass with `--path`.
If you want to point to a specific file, use `--config`.

If you do not pass `--envs`, commands that need environments will use the config file first and built-in defaults second.

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

If your team uses a custom ticket pattern:

```bash
gig find ABC-123 --path . --config gig.yaml
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

If you want to pass envs directly:

```bash
gig env status ABC-123 --path . --envs dev=dev,test=test,prod=main
```

If you already have a config file:

```bash
gig env status ABC-123 --path .
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
gig verify --ticket ABC-123 --from test --to main --path .
```

Optional JSON output:

```bash
gig verify --ticket ABC-123 --from test --to main --path . --format json
```

Batch verification from a file:

```bash
gig verify --ticket-file tickets.txt --from test --to main --path .
```

What it shows:

- one overall verdict: `safe`, `warning`, or `blocked`
- why the tool gave that verdict
- per-repo checks
- manual-review steps when risky files are detected

## `gig plan`

Use this when you want a clear, read-only promotion plan.

```bash
gig plan --ticket ABC-123 --from test --to main --path .
```

Optional JSON output:

```bash
gig plan --ticket ABC-123 --from test --to main --path . --format json
```

Batch planning from a file:

```bash
gig plan --ticket-file tickets.txt --from test --to main --path .
```

What it shows:

- per-repo verdict
- commits expected to move next
- environment status in the selected flow
- risk signals
- manual steps
- planned actions

## `gig manifest generate`

Use this when you want a release packet that people can copy into release communication or attach to a QA or client review.

Default Markdown output:

```bash
gig manifest generate --ticket ABC-123 --from test --to main --path .
```

Optional JSON output:

```bash
gig manifest generate --ticket ABC-123 --from test --to main --path . --format json
```

Batch packet generation from a file:

```bash
gig manifest generate --ticket-file tickets.txt --from test --to main --path .
```

What it shows:

- short release summary
- QA checklist
- client review notes
- release manager checklist
- per-repo details, risks, notes, and commits to include

## `gig doctor`

Use this when you want to check whether the workspace and config are in a healthy state.

```bash
gig doctor --path .
```

Optional JSON output:

```bash
gig doctor --path . --format json
```

What it checks:

- whether a config file was found
- whether repo catalog entries match real repos
- whether configured environment branches exist
- whether service, owner, and kind are filled in

## `gig version`

Use this when you want to confirm what build is installed.

```bash
gig version
```

## SCM Support Today

- Git and SVN working copies both support the current read-only CLI flow: `scan`, `find`, `inspect`, `env status`, `diff`, `verify`, `plan`, `manifest generate`, and `doctor`
- SVN branch comparison assumes a normal Subversion layout such as `trunk` and `branches/<name>`, or explicit branch paths in config and flags
- repository catalog entries currently map to detected repository roots, not subfolders inside a single monorepo

## Common Examples

### See what changed for a ticket

```bash
gig inspect ABC-123 --path /path/to/workspace
```

### Check whether `test` is behind `dev`

```bash
gig env status ABC-123 --path /path/to/workspace
```

### Check whether it is safe to move from `test` to `main`

```bash
gig verify --ticket ABC-123 --from test --to main --path /path/to/workspace
```

### Generate a release packet for people

```bash
gig manifest generate --ticket ABC-123 --from test --to main --path /path/to/workspace
```

### Generate JSON for CI or review tooling

```bash
gig plan --ticket ABC-123 --from test --to main --path /path/to/workspace --format json
```

### Check whether your config is good enough to trust

```bash
gig doctor --path /path/to/workspace
```

## Output Formats

- `human`
  easy to read in terminal and good for manual review
- `json`
  good for CI, scripts, and tooling
- `markdown`
  good for release packets and copy-paste communication

Output formats currently available on:

- `gig verify`: `human`, `json`
- `gig plan`: `human`, `json`
- `gig manifest generate`: `markdown`, `json`
- `gig doctor`: `human`, `json`

## Exit Codes

- `0`: success
- `1`: runtime failure
- `2`: usage error

## What Is Planned Next

Planned next CLI additions include:

- `gig plan --release <release-id>`
- richer Jira or deployment evidence
- multi-ticket release bundles
