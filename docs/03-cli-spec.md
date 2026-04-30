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
- accept direct command-palette input such as `ABC-123`, `verify ABC-123`, or `repo github:owner/name ABC-123`

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
```

Use this when you need a release verdict.

Rules:

- Remote mode tries provider-backed topology first.
- When the current branch looks like a promotion source, such as `staging`, `uat`, or `release/*`, it can become the default source branch.
- Local mode needs a promotion path unless a project, config, or interactive prompt supplies it.
- If topology is ambiguous, `gig` asks for explicit promotion intent in an interactive terminal. In scripts, pass `--from` and `--to`.
- Add `--envs` only when branch order cannot be inferred from source control or your explicit promotion intent.
- Add `--json` or `--format json` for automation.

## `gig packet`

```bash
gig packet ABC-123
gig packet ABC-123 --repo github:owner/name
gig packet ABC-123 --project payments
gig packet ABC-123 --from staging --to main
gig packet --ticket-file tickets.txt
gig packet --release rel-2026-04-09 --path .
```

Use this to export a release packet.

`gig manifest ...` and `gig manifest generate ...` still work for existing scripts.

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
| `--config` | Use optional team overrides. |

## Remote Repository Targets

```text
github:owner/name
gitlab:group/project
bitbucket:workspace/repo
azure-devops:org/project/repo
svn:https://svn.example.com/repos/app/branches/staging/ProductName
```

Use `--repo` for remote mode outside the checkout. Use `--path` for local fallback mode.

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
