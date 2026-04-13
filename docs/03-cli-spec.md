# CLI Guide

This page explains what each command does and when to use it.

If you are new, start with `gig --help` and then come back here.

## Start Here

### Authenticate to a live source-control provider

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

This uses the provider login flow under the hood today:

- `gh auth` for GitHub
- `glab auth` for GitLab
- `gig`-managed API token storage for Bitbucket, saved to macOS Keychain when available
- `az login` for Azure DevOps
- `gig`-managed stored credentials or `GIG_SVN_USERNAME` / `GIG_SVN_PASSWORD` for remote SVN

If you run a supported remote-backed command without an active session, `gig` will ask you to log in before continuing.
For Bitbucket, `GIG_BITBUCKET_EMAIL` and `GIG_BITBUCKET_API_TOKEN` can override stored credentials in CI or non-interactive environments.

### Show all commands

```bash
gig
```

Running `gig` with no subcommand opens the guided front door with:

- the current workarea, if one is selected
- the next inspect, verify, manifest, and assist commands
- a calmer summary block for scope and promotion context
- the optional `gig assist doctor` readiness check for the bundled DeerFlow sidecar
- the optional `gig assist setup` hint for the bundled DeerFlow sidecar

In an interactive terminal, `gig` also offers a quick-start prompt:

- if a current project exists, it offers quick actions, then asks for a ticket ID and can run `inspect`, `verify`, `manifest generate`, or `assist audit` immediately
- if no current project exists, it can accept a remote repo target, discover one from a provider, or switch to a saved workarea

For the full command list:

```bash
gig --help
```

Help screens are examples-first and grouped for scanning speed:

- `Start here`
- `Common flags`
- `Next commands`

Human `inspect`, `verify`, and `plan` outputs also lead with a compact summary block and a recommended next step before the per-repository detail.

### Get help for one command

```bash
gig inspect --help
gig verify --help
gig assist --help
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
| `gig workarea` | save and switch project defaults so later commands stay short |
| `gig scan` | find repos under a folder |
| `gig find` | find commits for one ticket |
| `gig inspect` | see the full ticket story across repos |
| `gig env status` | check where a ticket is present or behind across env branches |
| `gig diff` | compare one branch to another for a ticket |
| `gig verify` | get a quick `safe`, `warning`, or `blocked` result |
| `gig assist doctor` | check whether the bundled DeerFlow sidecar is configured, startable, and reachable |
| `gig assist setup` | bootstrap the bundled DeerFlow sidecar config and get the next start command |
| `gig assist audit` | turn one deterministic ticket audit bundle into an AI briefing |
| `gig assist release` | turn a saved or live release bundle into one AI release briefing |
| `gig assist resolve` | turn the active Git conflict state into one AI conflict briefing |
| `gig plan` | build a read-only promotion plan for people or CI |
| `gig snapshot create` | save a repeatable ticket baseline for audit and re-check |
| `gig manifest generate` | generate a Markdown or JSON release packet |
| `gig doctor` | check inferred topology, optional overrides, and repo health |
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
When you use `--repo github:owner/name`, `--repo gitlab:group/project`, `--repo bitbucket:workspace/repo`, `--repo azure-devops:org/project/repo`, or a branch-scoped SVN URL such as `--repo svn:https://svn.example.com/repos/app/branches/staging/ProductName`, `gig` can infer the release path directly from provider topology or standard SVN layout and use that as the default release path.

## Shared Workarea Behavior

Ticket-aware commands can also inherit defaults from a saved workarea.
If you start with a direct `--repo` command, `gig` can remember that repository as the current project automatically after the first successful remote run.

Example:

```bash
gig inspect ABC-123 --repo github:owner/name
gig inspect ABC-123
gig verify --ticket ABC-123
gig manifest generate --ticket ABC-123
```

What a workarea can remember:

- remote repo scope from `--repo`
- local fallback path from `--path`
- optional config file from `--config`
- default `--from` and `--to` branches
- default `--envs` mapping

Rules:

- explicit flags still win over the workarea
- `gig workarea use <name>` makes one workarea current
- `gig inspect`, `gig verify`, `gig plan`, `gig snapshot create`, `gig manifest generate`, and `gig assist audit` can use the current workarea without repeating `--repo`
- if a remote workarea has no local path, `gig` keeps a workarea home directory under the user config area for snapshots and other local state
- `gig workarea add --provider <provider>` can discover repositories from a logged-in account so you do not have to type the target manually
- interactive pickers accept either a number or filter text
- recently used workareas and recently selected repositories are promoted to the top of the picker

## Shared Remote Behavior

Commands that accept `--repo` can operate directly on a live remote repository:

```bash
gig inspect ABC-123 --repo github:owner/name
gig inspect ABC-123 --repo gitlab:group/project
gig inspect ABC-123 --repo bitbucket:workspace/repo
gig inspect ABC-123 --repo azure-devops:org/project/repo
gig inspect ABC-123 --repo svn:https://svn.example.com/repos/app/branches/staging/ProductName
gig verify --ticket ABC-123 --repo github:owner/name
gig manifest generate --ticket ABC-123 --repo github:owner/name
```

Today the remote live flow supports:

- GitHub repository targets in `github:owner/name` form
- GitLab repository targets in `gitlab:group/project` form
- Bitbucket repository targets in `bitbucket:workspace/repo` form
- Azure DevOps repository targets in `azure-devops:org/project/repo` form
- SVN repository targets in `svn:https://host/path/to/branch-scoped/repository` form
- auto-login through `gh auth login`, `glab auth login`, `gig login bitbucket`, or `az login`
- interactive or env-backed credential setup for remote SVN
- protected-branch detection for release inference
- direct ticket inspection and promotion checks without cloning first
- provider evidence sections for pull requests and deployments on GitHub, GitLab, Bitbucket, and Azure DevOps

## `gig workarea`

Use this when you want `gig` to remember one project so you do not have to keep typing repo targets and branch defaults.

```bash
gig inspect ABC-123 --repo github:owner/name
gig workarea add --provider github --use
gig workarea add payments --repo github:owner/name --from staging --to main --use
gig workarea list
gig workarea use payments
gig workarea show
```

The first command above is enough for many teams.
After it succeeds, `gig` can remember that remote repository as the current project automatically.

This command supports:

- `gig workarea add [<name>]`
- `gig workarea list`
- `gig workarea use [<name>]`
- `gig workarea show [<name>]`

If you run `gig workarea use` without a name, `gig` shows an interactive picker in the terminal. You can enter a number or type part of the project name to filter the list.
If you run `gig workarea add --provider github`, `gig workarea add --provider gitlab`, `gig workarea add --provider bitbucket`, or `gig workarea add --provider azure-devops`, `gig` discovers repositories from that logged-in account, promotes recent selections to the top, and lets you choose one with either a number or filter text.

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
- provider evidence such as pull requests and deployments when the remote provider can confirm them

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

## `gig assist resolve`

Use this when Git has already stopped on a conflict and you want an AI-written brief for the active conflict block before choosing a resolver action.

```bash
gig assist resolve --path . --ticket ABC-123 --audience release-manager
```

What it does:

- loads the deterministic `gig resolve` status and active conflict block for the current repository
- sends that bundle to DeerFlow for a concise QA, client, or release-manager briefing
- keeps `gig` as the source of truth for branch provenance, scope warnings, and risk signals

Important notes:

- this command is experimental
- it does not edit files, stage files, or continue the Git operation for you
- DeerFlow must be reachable locally or at the URL you pass with `--url`
- use `gig resolve start` when you are ready to apply a resolution

## `gig assist doctor`

Use this when you want a readiness check before writing DeerFlow config files.

```bash
gig assist doctor
gig assist doctor --path /path/to/gig-repo
gig assist doctor --url http://localhost:2026
```

What it does:

- finds the bundled `deer-flow/` directory in the current `gig` repo or the path you pass
- checks whether a local DeerFlow config already exists
- checks whether the current config has at least one active model plus the required credentials
- checks whether the local gateway responds at the URL you pass with `--url`
- reports the recommended next step before you try `assist audit`, `assist release`, or `assist resolve`

Important notes:

- this command is read-only
- a healthy gateway is helpful, but the main readiness result is about whether the sidecar is locally startable and configured
- use `gig assist setup` next when the report says `setup-required`

## `gig assist setup`

Use this when you want `gig` to bootstrap the bundled DeerFlow sidecar before trying AI briefings.

```bash
gig assist setup
gig assist setup --path /path/to/gig-repo
```

What it does:

- finds the bundled `deer-flow/` directory in the current `gig` repo or the path you pass
- creates `config.yaml` and any available `.env` templates from the bundled DeerFlow examples when they are missing
- reports whether Docker is available and prints the recommended next start command

Important notes:

- this command is local bootstrap only; it does not start long-running DeerFlow services for you
- run `gig assist doctor` first if you want a readiness report without creating files
- use this before `gig assist audit`, `gig assist release`, or `gig assist resolve` when the local sidecar is not configured yet

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

## `gig assist audit`

Use this when you want an AI-written release briefing from the same evidence that powers `inspect`, `verify`, and `manifest`.

```bash
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
```

Local mode:

```bash
gig assist audit --ticket ABC-123 --from test --to main --path . --audience client
```

What it does:

- builds a deterministic audit bundle from `gig` inspection and promotion logic
- sends that bundle to DeerFlow for a concise QA, client, or release-manager briefing
- keeps `gig` as the source of truth for commits, branches, risk signals, and verdicts

Important notes:

- this command is experimental
- supported audiences are `qa`, `client`, and `release-manager`
- DeerFlow must be reachable locally or at the URL you pass with `--url`
- AI output is additive explanation, not a replacement for the underlying `gig` verdict

## `gig assist release`

Use this when you want an AI-written release briefing for one named release.

```bash
gig assist release --release rel-2026-04-09 --path . --audience release-manager
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

Snapshot mode reads from:

```text
.gig/releases/<release-id>/snapshots/*.json
```

Live mode builds the same release bundle directly from `--ticket` or `--ticket-file` plus the selected local or remote repository scope.

What it does:

- loads all saved ticket snapshots for the release, or captures live ticket evidence into an in-memory release bundle
- builds one deterministic release bundle with release verdict, per-ticket rollups, and packet data
- sends that bundle to DeerFlow for a concise QA, client, or release-manager briefing

Important notes:

- this command is experimental
- supported audiences are `qa`, `client`, and `release-manager`
- DeerFlow must be reachable locally or at the URL you pass with `--url`
- AI output is additive explanation, not a replacement for the underlying `gig` release plan or verification

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

Use this when you want to check whether the workspace, inferred branch topology, and optional team overrides are in a healthy state.

```bash
gig doctor --path .
```

Optional JSON output:

```bash
gig doctor --path . --format json
```

What it checks:

- whether `gig` can scan the selected workspace
- whether inferred or configured environment branches exist
- whether optional repo catalog entries match real repos
- whether optional service, owner, and kind fields are filled in when you use overrides

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

- npm installs run `npm install -g @phamhungptithcm/gig@latest`
- `gig update vYYYY.MM.DD` maps the release tag to the matching npm package version automatically
- direct installs re-run the official installer in the current binary directory
- pinned versions are supported for npm and direct installs
- legacy Homebrew and Scoop installs are no longer published and should be reinstalled through npm or the direct installer

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

### Brief one saved release for a release manager

```bash
gig assist release --release rel-2026-04-09 --path /path/to/workspace --audience release-manager
```

### Brief one live remote release from a ticket file

```bash
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

### Brief one active conflict before resolving it

```bash
gig assist resolve --path /path/to/workspace/a-service --ticket ABC-123 --audience release-manager
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

### Check inferred topology or optional overrides

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
- `gig assist audit`: `human`, `json`
- `gig assist release`: `human`, `json`
- `gig assist resolve`: `human`, `json`
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
- better CI or deployment evidence around release bundles
