# CLI Spec

## Command Style

Root form:

```bash
gig <command> [flags]
```

The CLI should be easy to use in scripts and also easy to extend later for interactive mode.

## Shared Rules

- commands print human-readable output by default
- errors go to stderr
- successful commands exit with code `0`
- failures exit with non-zero codes
- read-only commands must never change repositories

## `gig scan`

Purpose:
Find repositories under a path.

```bash
gig scan --path .
```

Flags:

- `--path`: workspace path or repository path; defaults to `.`

Behavior:

- if `--path` is inside a repository, return that repository
- otherwise recursively scan descendants for supported SCM markers
- print repository name, SCM type, current branch when available, and root path

Output fields:

- repository name
- SCM type
- current branch if known
- repository path

## `gig find`

Purpose:
Find commits that match one ticket ID across all detected repositories.

```bash
gig find ABC-123 --path .
```

Arguments:

- `<ticket-id>`: required ticket key such as `ABC-123`

Flags:

- `--path`: workspace path or repository path; defaults to `.`

Behavior:

- discover repositories
- search commit history for commit messages containing the ticket ID
- group results by repository
- print short hash, subject, and containing branches when available

Output fields per commit:

- short hash
- subject
- branches when available

## `gig diff`

Purpose:
Show commits for one ticket that exist in the source branch but are still missing in the target branch.

```bash
gig diff --ticket ABC-123 --from dev --to test --path .
```

Flags:

- `--ticket`: required ticket key
- `--from`: required source branch
- `--to`: required target branch
- `--path`: workspace path or repository path; defaults to `.`

Behavior:

- discover repositories
- collect ticket-matching commits reachable from `--from`
- collect ticket-matching commits reachable from `--to`
- compare the two branches
- identify commits present in source but missing in target
- group results by repository

Output fields per repository:

- source branch name
- target branch name
- source commit count
- target commit count
- missing commits list

## `gig version`

Purpose:
Show the installed CLI version and build metadata.

```bash
gig version
```

Output:

- version string
- commit id when available
- build timestamp when available

## `gig promote`

Status:
Planned. Not implemented in the current code yet.

Purpose:
Build and optionally execute a safe promotion plan for one ticket.

```bash
gig promote ABC-123 --from test --to prod --dry-run --path .
```

Planned flags:

- `--path`: workspace path or repository path; defaults to `.`
- `--from`: source branch
- `--to`: target branch
- `--dry-run`: show plan only
- `--yes`: skip interactive confirmation in controlled automation

Planned behavior:

- collect commits for the ticket
- compare branches
- build a promotion plan
- show missing commits
- dry-run or execute after confirmation

## Exit Codes

- `0`: success
- `1`: runtime failure
- `2`: usage error
- `3`: partial success with warnings in a future multi-repo execution mode

## Output Format

Human-readable default output should follow these rules:

- group by repository
- keep headings short
- keep commit lines easy to scan
- show warnings clearly
- never hide write actions behind unclear text

Future JSON output should use the same service results, not separate command logic.
