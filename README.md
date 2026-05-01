# gig

<p align="center">
  <a href="docs/assets/gig-showcase.mp4">
    <img src="docs/assets/gig-showcase.gif" alt="gig terminal demo showing ticket inspect, verify, and packet output" width="860">
  </a>
</p>

<p align="center">
  <strong>Ticket-to-release confidence from one terminal command.</strong>
</p>

<p align="center">
  <a href="https://phamhungptithcm.github.io/gig/">Docs</a> ·
  <a href="docs/19-quickstart.md">Quick start</a> ·
  <a href="docs/25-demo-guide.md">Demo guide</a> ·
  <a href="docs/troubleshooting.md">Troubleshooting</a>
</p>

`gig` is a remote-first release audit CLI for the question that slows down
release day:

> Did every change for this ticket actually make it into the release?

It follows the ticket across commits, branches, pull requests, deployments,
checks, linked work, and release notes. Then it gives you a deterministic
answer: `safe`, `warning`, or `blocked`.

```bash
gig
# ask gig > repo payments
# ask gig > ABC-123
# ask gig > verify
# ask gig > packet
```

## Why Teams Use It

- One command answers the release question before QA or client review.
- Remote-first provider access means no clone is required for GitHub, GitLab,
  Bitbucket, Azure DevOps, or remote SVN audits.
- Zero-config local mode still works from an existing Git or SVN checkout.
- Smart suggestions print exact next commands instead of vague advice.
- Long human commands show a small loading bar on stderr so users know `gig` is still working.
- Missing tools and auth failures explain what is needed and how to install it.
- Human output is terminal-friendly; JSON output stays clean for CI.

## Install

macOS or Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
```

On macOS and Linux, the installer now mirrors the Windows installer behavior:
when it has to use a user-local install directory, it adds that directory to
your zsh, bash, fish, or profile PATH setup and tells you to open a new terminal.
The installer verifies the downloaded binary directly; open a new terminal
before running `gig version` if it added a new PATH entry.

Windows PowerShell:

```powershell
irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1 | iex
gig version
```

npm, for teams that distribute CLIs through Node:

```bash
npm install -g @hunpeolabs/gig
gig version
```

Pin a release when you need a reproducible rollout:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh -s -- --version vYYYY.M.MICRO
```

Release tags use the same CalVer version as npm with a `v` prefix, for example `v2026.5.0`.
Older padded date tags such as `v2026.04.17` remain installable when you pin an existing release.

Refresh later with:

```bash
gig update
```

## Start Fast

Inside a repo with a supported `origin`, the shortest useful path is:

```bash
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

From anywhere, let the prompt find or remember the remote target:

```bash
gig setup --provider github
gig login github
gig repo payments
gig
# ask gig > repo payments
# ask gig > ABC-123
# ask gig > verify
# ask gig > packet
```

Scripts and CI can still use explicit canonical targets with
`--repo github:owner/name`.

Run only `gig` when you are not sure what to type. The front door detects the
current checkout, shows provider status, and suggests the shortest next command.
You can type `ABC-123`, `verify ABC-123`, `packet ABC-123`, or
`repo payments` directly into it. `repo` opens provider discovery, short-name
search checks saved, recent, and logged-in provider repos, and normal URLs or
remotes are normalized for you. The prompt stays open after
each command; type `exit` or `quit` when you are done. Suggestions are ranked as
a workflow, so `now`, `verify`, `packet`, and `save` keep the next move obvious.
Inside that session, `verify`, `packet`, `explain`, `next`, and `last` reuse the
last ticket and scope. Short aliases also work: `i ABC-123`, `v`, `p`, `r`,
`?`, `gh owner/name`, `gl group/project`, `bb workspace/repo`, and
`ado org/project/repo`. Use `save payments` once a repo is remembered so later
sessions can start with `use payments`, `ABC-123`, `verify`, and `packet`. When
a `run?` row appears, press Enter to run the suggested next command.

Read-only commands do not start interactive provider login. If auth is missing,
`gig` prints the exact command to run, for example:

```bash
gig login github
```

## What You Get

`inspect` builds the ticket story:

```bash
gig inspect ABC-123 --repo github:owner/name
```

`verify` compares source and target branches and returns a release verdict:

```bash
gig verify ABC-123 --repo github:owner/name
gig verify ABC-123 --out verify.xlsx
gig verify ABC-123 --out verify.csv
```

`packet` exports the release packet for QA, release managers, or client review:

```bash
gig packet ABC-123 --repo github:owner/name
gig packet ABC-123 --out release-packet.xlsx
gig packet ABC-123 --format csv --out release-packet/
gig packet ABC-123 --json
```

`manifest` remains as a compatibility alias, but `packet` is the clearer command
for new workflows.

## Release Exports

Use XLSX when the artifact is meant for release managers, QA, engineering
leads, or compliance review. It creates a professional workbook with stable
sheets, filters, frozen headers, and formula-injection-safe cells.

Use CSV when another spreadsheet, reporting, or compliance system needs import
tables. `verify --out verify.csv` writes one practical verification table.
Release packets contain multiple tables, so CSV packet export writes a directory.

Use JSON for automation and CI contracts:

```bash
gig verify ABC-123 --format json
gig packet ABC-123 --format json
```

Verification XLSX sheets: Summary, Decision, Risks, Missing Changes, Commits,
Manual Steps, Evidence, Metadata.

Release packet XLSX sheets: Cover, Release Decision, Scope, Risks, Missing
Changes, Commits, Manual Steps, Verification, Approvals, Evidence, Metadata.

Release packet CSV directory files: `summary.csv`, `release-decision.csv`,
`scope.csv`, `risks.csv`, `missing-changes.csv`, `commits.csv`,
`manual-steps.csv`, `verification.csv`, `approvals.csv`, `evidence.csv`, and
`metadata.csv`.

Release decisions are conservative: `blocked` for missing release changes or
failed required checks, `needs_review` for medium risks, manual steps, ambiguous
topology, or incomplete evidence, `ready` only when evidence is complete and no
blocking risk remains, and `unknown` when setup, auth, provider, or config access
prevents a reliable answer.

## Provider Coverage

Remote target forms:

```text
github:owner/name
gitlab:group/project
bitbucket:workspace/repo
azure-devops:org/project/repo
svn:https://svn.example.com/repos/app/branches/staging/ProductName
```

The canonical forms above are stable for scripts. Humans can also paste normal
provider URLs and remotes, such as `https://github.com/owner/name`,
`git@github.com:owner/name.git`, `https://dev.azure.com/org/project/_git/repo`,
or a plain SVN URL.

| Provider | Coverage today |
| --- | --- |
| GitHub | Deep release evidence: PRs, deployments, checks, linked issues, releases |
| GitLab | Deep release evidence: merge requests, deployments, checks, linked issues, releases |
| Bitbucket | Basic release evidence: pull requests, deployments, branching model |
| Azure DevOps | Deep release evidence: pull requests, deployments, checks, linked work items |
| Remote SVN | Audit topology only: branch and trunk discovery |

When protected branches are clear, `gig` infers source and target topology.
When topology is ambiguous, it stops and prints the copyable command you can run
with `--from` and `--to`.

## Dependency UX

`gig setup` checks required tools without changing your machine:

```bash
gig setup --provider github
```

If something is missing, the diagnostic includes why it is needed, install
commands for macOS, Windows, and Linux, and the next `gig login <provider>`
command.

Install is opt-in:

```bash
gig setup --provider github --install-missing
```

`gig` asks before running install commands. It does not silently install system
tools during read-only commands such as `inspect`, `verify`, or `packet`.

## Local And CI

Local fallback stays available:

```bash
gig ABC-123 --path .
gig verify ABC-123 --path . --from staging --to main
gig project add local --path . --from staging --to main --use
```

CI and logs stay plain:

```bash
NO_COLOR=1 gig verify ABC-123 --repo github:owner/name
gig verify ABC-123 --repo github:owner/name --json
```

Human output wraps for normal terminal widths. JSON output is free of ANSI color
and visual-only formatting.

## Optional AI Layer

`gig` remains the source of truth. The AI layer is optional and works from the
release evidence `gig` already collected.

```bash
gig explain ABC-123
gig resume
gig ask "what is still blocked?"
gig ask "what changed since the last brief?"
```

For release-day diagnostics, set:

```bash
export GIG_DIAGNOSTICS_FILE=/tmp/gig-diagnostics.jsonl
```

That file captures structured auth and topology events without turning normal
terminal output into log noise.

## Docs And Demo

- [Quick Start](docs/19-quickstart.md)
- [CLI Spec](docs/03-cli-spec.md)
- [Demo Guide](docs/25-demo-guide.md)
- [Portfolio Guide](docs/26-portfolio-guide.md)
- [Troubleshooting](docs/troubleshooting.md)
- [Docs site](https://phamhungptithcm.github.io/gig/)

## Benchmark

Compare manual repo-by-repo audit work against one `gig` command on the same
synthetic workspace:

```bash
./scripts/benchmark-release-audit.sh --runs 5
```

![gig benchmark snapshot](docs/assets/release-audit-benchmark.svg)

Sample run on April 17, 2026 on `Darwin arm64` / macOS `26.2`:

| Scenario | Avg ms | Commands | Human steps |
| --- | ---: | ---: | ---: |
| manual git loop | 63 | 6 | 7 |
| `gig verify` | 354 | 1 | 1 |

The honest claim is workflow compression, deterministic verdicts, and less
repo-by-repo review churn. On the synthetic local workspace, the manual loop is
still faster in raw elapsed time; on real release-day work, people also have to
read, reconcile, and explain the result.

Resume-friendly proof point:

> Built a remote-first release audit CLI that reduced a representative 2-repo
> release check from 6 terminal commands / 7 manual steps to 1 command / 1 step
> while producing a deterministic release verdict.
