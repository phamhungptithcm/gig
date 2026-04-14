# CLI Guide

This page explains the command surface in the order most users actually need it.

If you are new, start with:

```bash
gig --help
```

## The Main Workflow

For most teams, `gig` should feel like this:

```bash
gig
gig login github
gig ABC-123 --repo github:owner/name
gig verify ABC-123 --repo github:owner/name
gig manifest ABC-123 --repo github:owner/name
```

When users run `gig` in a real terminal, the front door should behave like a guided picker:

- default to GitHub-first onboarding
- let users choose with `↑/↓` and `Enter` instead of typing numbers
- keep local-folder fallback visible
- surface saved projects without forcing workarea setup first

The front door also accepts direct palette input such as:

- `ABC-123`
- `inspect ABC-123`
- `verify ABC-123`
- `manifest ABC-123`
- `repo github:owner/name ABC-123`
- `login github`

## Command Groups

| Command | Use it when you want to... |
| --- | --- |
| `gig` | open the guided front door |
| `gig login` | authenticate to a live provider |
| `gig inspect` | see the full ticket story |
| `gig verify` | get a `safe`, `warning`, or `blocked` verdict |
| `gig manifest` | export a release packet |
| `gig workarea` | remember repo scope and defaults for repeat use |
| `gig plan` | build a read-only promotion plan |
| `gig snapshot create` | save a repeatable ticket baseline |
| `gig assist *` | add an optional AI briefing layer |
| `gig scan`, `gig find`, `gig env status`, `gig diff` | use lower-level inspection tools |
| `gig doctor` | check repo health, overrides, and inference |
| `gig resolve *` | inspect or resolve active Git conflicts |
| `gig update`, `gig version` | manage the installed CLI |

## Core Commands

### `gig`

```bash
gig
```

Use this to open the guided front door.
In an interactive terminal, `gig` should let users pick the next useful action with `↑/↓` and `Enter` based on their current project, saved workareas, or GitHub discovery flow.

### `gig login`

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

Use this once per provider before running remote-backed commands.

### `gig inspect`

```bash
gig ABC-123
gig ABC-123 --repo github:owner/name
gig inspect ABC-123
gig inspect ABC-123 --workarea payments
gig ABC-123 --path .
```

Use this when you need the full ticket audit:

- repositories touched
- commit evidence
- branch presence
- risk hints and follow-up fixes

### `gig verify`

```bash
gig verify ABC-123
gig verify ABC-123 --repo github:owner/name
gig verify --ticket-file tickets.txt --repo github:owner/name
gig verify --release rel-2026-04-09 --path .
```

Use this when you need a release decision instead of raw evidence.
Add `--from` and `--to` only when `gig` cannot infer the promotion path.

### `gig manifest`

```bash
gig manifest ABC-123
gig manifest ABC-123 --repo github:owner/name
gig manifest generate ABC-123
```

Use this to generate a release packet for QA, release review, or automation.
The older `gig manifest generate ...` form still works, but `gig manifest ...` is the main command to remember.

### `gig workarea`

```bash
gig workarea add payments --repo github:owner/name --from staging --to main --use
gig workarea list
gig workarea use payments
gig workarea show
```

Use a workarea when you want `gig` to remember repo scope and defaults so later commands can stay short.

### `gig update`

```bash
gig update
gig update v2026.04.09
```

Use this to refresh the installed CLI.
If your npm install still returns `404`, the first package publish has not completed yet; use the direct installer until it does.

## Optional AI Layer

### Readiness And Setup

```bash
gig assist doctor
gig assist setup
```

Use these before the first AI-assisted flow to confirm the bundled DeerFlow sidecar is configured and reachable.

### Ticket Briefing

```bash
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
gig assist audit --ticket ABC-123 --repo github:owner/name --audience client
gig assist audit --ticket ABC-123 --repo github:owner/name --audience release-manager
```

### Release Briefing

```bash
gig assist release --release rel-2026-04-09 --path . --audience release-manager
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

### Conflict Briefing

```bash
gig assist resolve --path . --ticket ABC-123 --audience release-manager
```

These commands are optional and experimental.
They explain the deterministic `gig` bundle; they do not replace it.

## Lower-Level Commands

Use these when you need a narrower tool than `inspect` or `verify`:

- `gig scan --path .`
  find repositories under a local folder
- `gig find ABC-123 --path .`
  list raw commits for a ticket
- `gig env status ABC-123 --path .`
  see where a ticket is present or behind across environment branches
- `gig diff --ticket ABC-123 --from dev --to test --path .`
  compare one branch to another for a ticket
- `gig plan ABC-123 --repo github:owner/name`
  build a read-only promotion plan
- `gig snapshot create --ticket ABC-123 --path .`
  save a repeatable audit baseline

## Conflict Commands

Use these only when Git has already stopped on a conflict:

```bash
gig resolve status --path .
gig resolve start --path .
```

`gig resolve start` can help with supported text conflicts, but it does not continue the Git operation or create a commit for you.

## Shared Behavior

- human-readable output is the default
- add `--format json` when you want automation-friendly output
- explicit flags win over workarea defaults
- `gig` auto-detects `gig.yaml`, `gig.yml`, `.gig.yaml`, and `.gig.yml`
- commands are read-only by default, except for the active conflict resolver

## Remote Repository Targets

Use `--repo` for live remote access:

- `github:owner/name`
- `gitlab:group/project`
- `bitbucket:workspace/repo`
- `azure-devops:org/project/repo`
- `svn:https://svn.example.com/repos/app/branches/staging/ProductName`

Use `--path` for local fallback mode.

## Practical Rule

Start with:

1. `gig`
2. `gig login <provider>`
3. `gig <ticket-id>`
4. `gig verify <ticket-id>`
5. `gig manifest <ticket-id>`

Reach for the rest only when the main workflow does not answer the question fast enough.
