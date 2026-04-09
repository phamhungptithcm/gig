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
gig resolve --help
gig update --help
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
| `gig snapshot create` | save a repeatable ticket baseline for audit and re-check |
| `gig manifest generate` | generate a Markdown or JSON release packet |
| `gig doctor` | check config coverage, env mapping, and repo catalog health |
| `gig resolve status` | inspect the current Git conflict state in one repository |
| `gig resolve start` | walk supported Git text conflicts with keyboard actions |
| `gig update` | refresh the installed CLI to the latest release or a specific version |
| `gig version` | confirm what build you installed |

## Shared Rules

- commands print human-readable output by default unless you ask for JSON
- errors go to stderr
- successful commands exit with code `0`
- usage errors return code `2`
- runtime failures return code `1`
- commands are read-only by default
- `gig resolve start` writes the working tree for the active conflicted repository, but it does not continue the Git operation or create a commit for you
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

## `gig resolve status`

Use this when Git has already stopped on a conflict and you want a quick picture of what is unresolved.

```bash
gig resolve status --path .
```

Optional JSON output:

```bash
gig resolve status --path . --format json
```

Optional ticket scoping:

```bash
gig resolve status --path . --ticket ABC-123
```

What it shows:

- the active Git operation type such as merge, rebase, or cherry-pick
- the current side and incoming side with branch, commit, and ticket context when available
- unresolved files
- which files `gig resolve start` can handle directly
- which files still need manual resolution

## `gig resolve start`

Use this when Git has already stopped on a supported text conflict and you want a keyboard-first resolver.

```bash
gig resolve start --path .
```

Optional ticket scoping:

```bash
gig resolve start --path . --ticket ABC-123
```

What it does:

- starts at the first supported conflict block
- lets you accept current, incoming, or both in either order
- lets you open the file in your editor, undo the last local choice, and stage a resolved file
- never runs `git merge --continue`, `git rebase --continue`, `git cherry-pick --continue`, or `git commit` for you

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

Release verification from saved snapshots:

```bash
gig verify --release rel-2026-04-09 --path .
```

What it shows:

- one overall verdict: `safe`, `warning`, or `blocked`
- why the tool gave that verdict
- per-repo checks
- manual-review steps when risky files are detected
- or a release-wide verification bundle when `--release` is used

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

## `gig snapshot create`

Use this when you want to save a repeatable baseline before release review or promotion.

```bash
gig snapshot create --ticket ABC-123 --from test --to main --path . --output .gig/snapshots/abc-123.json
```

Attach the snapshot to a named release and let `gig` choose the default release path:

```bash
gig snapshot create --ticket ABC-123 --from test --to main --path . --release rel-2026-04-09
```

Optional JSON output to stdout:

```bash
gig snapshot create --ticket ABC-123 --from test --to main --path . --format json
```

What it shows:

- capture time and tool version
- the inspection baseline for the ticket
- the current promotion plan
- the current verification result
- an optional JSON artifact path when `--output` is used

## `gig plan --release`

Use this when you want a release-level plan built from saved ticket snapshots.

```bash
gig plan --release rel-2026-04-09 --path .
```

This command reads snapshots from:

```text
.gig/releases/<release-id>/snapshots/*.json
```

Optional JSON output:

```bash
gig plan --release rel-2026-04-09 --path . --format json
```

What it shows:

- release-level verdict
- ticket baseline summary across the release
- repository roll-up across all saved ticket snapshots
- shared blockers, manual steps, and planned actions

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

Release bundle generation from saved snapshots:

```bash
gig manifest generate --release rel-2026-04-09 --path .
```

What it shows:

- short release summary
- QA checklist
- client review notes
- release manager checklist
- per-repo details, risks, notes, and commits to include
- or a release packet bundle when `--release` is used

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

## `gig update`

Use this when you want to refresh the installed CLI.

Update to the latest release:

```bash
gig update
```

Install a specific version:

```bash
gig update v2026.04.09
```

What it does:

- direct installs re-run the official installer in the current binary directory
- Homebrew installs run `brew update` and `brew upgrade gig-cli`
- Scoop installs run `scoop update gig-cli`
- pinned versions are supported for direct installs, not for Homebrew or Scoop

## `gig version`

Use this when you want to confirm what build is installed.

```bash
gig version
```

## SCM Support Today

- Git and SVN working copies both support the current read-only CLI flow: `scan`, `find`, `inspect`, `env status`, `diff`, `verify`, `plan`, `snapshot create`, `manifest generate`, and `doctor`
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

### Save a release baseline before promotion

```bash
gig snapshot create --ticket ABC-123 --from test --to main --path /path/to/workspace --output .gig/snapshots/abc-123.json
```

### Save a ticket into a named release

```bash
gig snapshot create --ticket ABC-123 --from test --to main --path /path/to/workspace --release rel-2026-04-09
```

### Review one saved release bundle

```bash
gig plan --release rel-2026-04-09 --path /path/to/workspace
```

### Re-run verification for one saved release

```bash
gig verify --release rel-2026-04-09 --path /path/to/workspace
```

### Generate one saved release bundle for people

```bash
gig manifest generate --release rel-2026-04-09 --path /path/to/workspace
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

### Refresh the installed CLI

```bash
gig update
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
- `gig snapshot create`: `human`, `json`
- `gig manifest generate`: `markdown`, `json`
- `gig doctor`: `human`, `json`

## Exit Codes

- `0`: success
- `1`: runtime failure
- `2`: usage error

## What Is Planned Next

Planned next CLI additions include:

- richer Jira or deployment evidence
- multi-ticket release bundles
