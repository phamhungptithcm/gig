# CLI Guide

This guide explains the commands in the order most users need them.

Start from inside the Git checkout for the repo you are auditing:

```bash
gig
gig login
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

When the current Git `origin` points at GitHub, GitLab, Bitbucket, or Azure DevOps, `gig` infers the remote repo target automatically. Use `--repo` only when you are outside that checkout, scripting against another repo, or overriding the inferred target.

## Command Groups

| Command | Use it when you want to |
| --- | --- |
| `gig` | Open the guided front door. |
| `gig login` | Authenticate to GitHub, GitLab, Bitbucket, Azure DevOps, or SVN. |
| `gig inspect` or `gig ABC-123` | Inspect one ticket. |
| `gig verify` | Get a `safe`, `warning`, or `blocked` verdict. |
| `gig packet` | Generate a release packet. |
| `gig project` | Save repo and branch defaults for repeated work. |
| `gig plan` | See a more detailed read-only promotion plan. |
| `gig snapshot create` | Save a repeatable audit baseline. |
| `gig explain`, `gig ask`, `gig resume` | Add optional AI explanation on top of deterministic evidence. |
| `gig repos`, `gig commits`, `gig where`, `gig diff` | Use lower-level local or advanced inspection tools. |
| `gig doctor` | Check repo health, overrides, diagnostics, and inference. |
| `gig setup` | Check or install missing local tools with confirmation. |
| `gig resolve` | Inspect or help resolve active Git conflicts. |
| `gig update`, `gig version` | Manage the installed CLI. |

Compatibility aliases remain available: `manifest` for `packet`, `workarea` for `project`, `scan` for `repos`, `find` for `commits`, and `env status` for `where`.

## `gig`

```bash
gig
```

Opens the guided front door.

The front door can:

- detect the current Git or SVN checkout
- show provider status and capability coverage
- resume saved project or AI context
- accept direct command-palette input such as `ABC-123`, `verify ABC-123`, `repo payments`, `gh owner/name`, or a pasted repo URL
- resolve human-friendly repo input: `repo`, `repo payments`, `gh owner/name`, `gl group/project`, `bb workspace/repo`, `ado org/project/repo`, or pasted provider/SVN URLs
- save the remembered repo scope with `save payments`, then switch later with `use payments`
- rank suggested next steps as a short workflow, for example `now`, `verify`, `packet`, and `save`
- keep the prompt open after completed or failed commands until you type `exit` or `quit`
- remember the last ticket and scope inside the session, so `verify`, `packet`, `explain`, `next`, and `last` can stay short
- support prompt aliases: `i`, `v`, `p`, `r`, `?`, `last`, and provider aliases such as `gh`, `gl`, `bb`, `ado`, and `svn`
- let Enter run the suggested next command when a `run?` hint is shown
- show a small loading bar below the command for long-running human output, without changing JSON stdout
- show confidence hints such as remembered scope, source branch, and release target

## `gig repo`

```bash
gig repo payments
gig repo github:owner/name
gig repo gh owner/name
gig repo https://github.com/owner/name
```

Resolves a human-friendly repository name, provider target, URL, or remote into
the canonical target `gig` uses internally. It records the selection as a recent
repo and prints the `gig project add ... --use` command when the user wants to
save it for shorter future commands.

Use `gig repo` outside the prompt when you want the same resolver without
opening a long-lived session. Inside the prompt, `repo payments` is still the
shortest form.

## `gig login`

```bash
gig login
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

Use this before remote-backed commands. If the provider is omitted, `gig` infers from the current checkout when safe; otherwise it asks.

Read-only commands such as `gig inspect`, `gig verify`, and `gig packet` do not launch interactive provider login. When auth or a provider CLI is missing, they stop, explain what is needed, and print the exact `gig login <provider>` command to run next.

## `gig setup`

```bash
gig setup
gig setup --provider github
gig setup --provider all --install-missing
```

Use this to check local tool readiness before the first remote audit.

`gig setup` never installs tools by default. `--install-missing` asks for confirmation before running install commands; `--yes` is only for explicit automation.

Provider tools:

- GitHub: `gh`
- GitLab: `glab`
- Azure DevOps: `az`
- SVN: `svn`

Bitbucket uses API-token credentials rather than a provider CLI.

Provider coverage:

| Provider | Coverage |
| --- | --- |
| GitHub | Deep release evidence: PRs, deployments, checks, linked issues, releases |
| GitLab | Deep release evidence: merge requests, deployments, checks, linked issues, releases |
| Bitbucket | Basic release evidence: pull requests, deployments, branching model |
| Azure DevOps | Deep release evidence: pull requests, deployments, checks, linked work items |
| SVN | Audit topology only: branch and trunk discovery |

## `gig inspect`

```bash
gig ABC-123
gig inspect ABC-123
gig ABC-123 --repo github:owner/name
gig ABC-123 --project payments
gig ABC-123 --path .
```

Use this when you need the full ticket audit.

Typical output includes:

- repositories touched
- commits and branches
- provider evidence when supported
- declared dependencies
- risk hints

## `gig verify`

```bash
gig verify ABC-123
gig verify ABC-123 --repo github:owner/name
gig verify ABC-123 --project payments
gig verify ABC-123 --from staging --to main
gig verify --ticket-file tickets.txt
gig verify --release rel-2026-04-09 --path .
gig verify ABC-123 --out verify.xlsx
gig verify ABC-123 --out verify.csv
```

Use this when you need a release verdict.

Rules:

- Remote mode tries provider-backed topology first.
- When the current branch looks like a promotion source, such as `staging`, `uat`, or `release/*`, it can become the default source branch.
- Local mode needs a promotion path unless a project, config, or interactive prompt supplies it.
- If topology is ambiguous, `gig` asks for explicit promotion intent in an interactive terminal. In scripts, pass `--from` and `--to`.
- Add `--envs` only when branch order cannot be inferred from source control or your explicit promotion intent.
- Add `--json` or `--format json` for automation.
- Add `--out verify.xlsx` for a shareable verification workbook.
- Add `--out verify.csv` for one import-friendly verification table.

Verification XLSX sheets:

- Summary
- Decision
- Risks
- Missing Changes
- Commits
- Manual Steps
- Evidence
- Metadata

## `gig packet`

```bash
gig packet ABC-123
gig packet ABC-123 --repo github:owner/name
gig packet ABC-123 --project payments
gig packet ABC-123 --from staging --to main
gig packet --ticket-file tickets.txt
gig packet --release rel-2026-04-09 --path .
gig packet ABC-123 --out release-packet.xlsx
gig packet ABC-123 --format csv --out release-packet/
```

Use this to export a release packet.

`gig manifest ...` and `gig manifest generate ...` still work for existing scripts.

XLSX is the best packet format for release managers, QA, engineering leads, and
compliance reviewers. CSV is for spreadsheet or reporting imports. JSON remains
the automation format.

Release packet XLSX sheets:

- Cover
- Release Decision
- Scope
- Risks
- Missing Changes
- Commits
- Manual Steps
- Verification
- Approvals
- Evidence
- Metadata

Release packet CSV export writes a directory because the packet contains
multiple tables:

- `summary.csv`
- `release-decision.csv`
- `scope.csv`
- `risks.csv`
- `missing-changes.csv`
- `commits.csv`
- `manual-steps.csv`
- `verification.csv`
- `approvals.csv`
- `evidence.csv`
- `metadata.csv`

Release decision values are:

- `ready`: no blocking risks and evidence is complete
- `needs_review`: medium risks, manual steps, ambiguous topology, or incomplete evidence
- `blocked`: missing release changes or a required check failed
- `unknown`: setup, auth, provider, or config access prevented a reliable answer

## `gig project`

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
gig project list
gig project use payments
gig project show
```

Use a project when you want short repeated commands from outside the checkout:

```bash
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

`gig workarea ...` remains available as a compatibility alias.

## `gig plan`

```bash
gig plan ABC-123
gig plan ABC-123 --repo github:owner/name
gig plan ABC-123 --project payments
gig plan --release rel-2026-04-09 --path .
```

Use this when you need more detail than `verify`. Most users should learn `verify` first.

## Lower-Level Commands

```bash
gig repos --path .
gig commits ABC-123 --path .
gig where ABC-123 --project payments
gig diff --ticket ABC-123 --from dev --to test --path .
gig snapshot create ABC-123 --path . --from staging --to main
```

Use these when you are debugging, building automation, or operating in local fallback mode.

## Optional AI Commands

```bash
gig assist doctor
gig assist setup
gig explain ABC-123
gig explain ABC-123 --audience qa
gig assist release --release rel-2026-04-09 --audience release-manager
gig ask "what is still blocked?"
gig resume
```

AI assist is optional. It summarizes deterministic `gig` bundles and can follow up using read-only `gig` tools.

## Shared Flags

| Flag | Meaning |
| --- | --- |
| `--repo` | Use a live remote target when it cannot be inferred from the current checkout. |
| `--path` | Use local Git/SVN fallback. |
| `--project`, `--workarea` | Reuse saved repo and branch defaults. |
| `--from`, `--to` | Set the promotion path when inference cannot. |
| `--envs` | Override environment-to-branch mapping. |
| `--json` | Print JSON for automation on supported commands. |
| `--format json` | Print JSON for automation. |
| `--format xlsx --out file.xlsx` | Write a shareable workbook for supported release commands. |
| `--format csv --out file.csv` | Write a single import table where the command has one natural table. |
| `--format csv --out directory/` | Write one CSV file per release-packet table. |
| `--out` | Infer export format from `.xlsx`, `.csv`, `.json`, or a CSV directory path. |
| `--config` | Use optional team overrides. |

## Remote Repository Targets

```text
github:owner/name
gitlab:group/project
bitbucket:workspace/repo
azure-devops:org/project/repo
svn:https://svn.example.com/repos/app/branches/staging/ProductName
```

`--repo` also accepts normal provider URLs and remotes such as `https://github.com/owner/name`, `git@github.com:owner/name.git`, `https://dev.azure.com/org/project/_git/repo`, and plain SVN URLs such as `https://svn.example.com/repos/app/branches/staging/ProductName`.

Inside `gig`, humans can avoid these canonical forms most of the time:

```text
repo payments
gh owner/name
gl group/project
bb workspace/repo
ado org/project/repo
svn https://svn.example.com/repos/app/branches/staging/ProductName
save payments
```

Use `--repo` for scripts and remote mode outside the checkout. Use `--path` for local fallback mode.

## Diagnostics

Set this when you need support or CI traces:

```bash
export GIG_DIAGNOSTICS_FILE=/path/to/gig-diagnostics.jsonl
```

Diagnostics are useful for auth, provider API, and topology inference issues.

## Next

- [Remote Repositories](remote-repositories.md)
- [Projects](workareas.md)
- [Troubleshooting](troubleshooting.md)
