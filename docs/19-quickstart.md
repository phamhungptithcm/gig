# Quick Start

This page is the first five minutes with `gig`.

Remember this path:

`install -> login -> inspect -> verify -> packet`

## 1. Install

Use the direct installer when you want the canonical release binary and self-update path:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
```

On macOS and Linux, the installer adds the selected user-local install
directory to your shell profile when it is not already on `PATH`, matching the
Windows installer habit of making `gig` available from the next terminal. It
handles zsh, bash, fish, and `.profile` fallback files. The installer prints the
installed version directly; if it added a PATH entry, open a new terminal before
running `gig version`.

Pin a version for team rollout:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh -s -- --version v2026.5.0
```

Release tags use the same CalVer version as npm with a `v` prefix. The `MICRO`
number starts at `0` each month and increments for additional releases.

Use npm when your team distributes CLI tools through npm:

```bash
npm install -g @hunpeolabs/gig
gig version
```

Expected success:

- `gig version` prints the installed version.
- `gig --help` prints the command summary.

Common failure:

- `gig: command not found`

Next action:

- Reopen the terminal or add the install directory to `PATH`.

## 2. Start In The Repo

```bash
cd /path/to/repo
git remote get-url origin
git branch --show-current
```

Use the checkout you are already working in. For GitHub, GitLab, Bitbucket, and Azure DevOps origins, `gig` can infer the remote repo target from `origin`.

Expected success:

- `origin` points at the provider repo you want to audit.
- the current branch is the release source when you are checking a promotion branch such as `staging`, `uat`, or `release/*`.

Common failure:

- the repo has no `origin`, or `origin` points at an unsupported host.

Next action:

- Use `--repo github:owner/name` for remote mode from anywhere.
- Use `--path .` for local fallback mode.

## 3. Open The Front Door

```bash
gig
```

Use this when you are not sure which command or repo target to type.

Expected success:

- `gig` shows a focused header, current checkout or provider status, suggested next commands, and a small command prompt.
- Suggestions are ranked as a workflow, such as `now`, `verify`, `packet`, and `save`, so you can keep moving without memorizing flags.
- In an interactive terminal, use `↑/↓`, `Enter`, or direct text such as `ABC-123`.
- The prompt stays open after a command completes or fails; type `exit` or `quit` when you are done.
- After one ticket command, `verify`, `packet`, `explain`, `next`, and `last` reuse the remembered ticket and scope.
- Use prompt aliases when you want speed: `i ABC-123`, `v`, `p`, `r`, `?`, `gh owner/name`, `gl group/project`, `bb workspace/repo`, and `ado org/project/repo`.
- Type `repo payments` to search saved, recent, and logged-in provider repos by short name.
- Paste normal URLs or remotes instead of memorizing canonical targets.
- Type `save payments` after a repo is remembered so future sessions can use `use payments`, `ABC-123`, `verify`, and `packet`.
- When a `run?` row appears, press Enter to run that suggested command immediately.
- If a remote scan or release check takes time, `gig` shows a small loading bar below the command so you know it is still working. JSON stdout stays clean.

Common failure:

- The UI cannot detect a repo or provider.

Next action:

- Type `repo`, `repo payments`, `gh owner/name`, or paste the repo URL. If auth is missing, run `gig login` first.
- Outside the prompt, `gig repo payments` uses the same resolver and prints a save command for the selected target.

## 4. Log In

```bash
gig setup --provider github
gig login
```

Use setup to check local provider tools, then log in once per provider. If you are already inside a supported Git checkout, `gig` tries to infer the matching provider. Otherwise it asks which provider to use.

Provider-specific forms also work:

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login azure-devops
gig login svn
```

Expected success:

- `gig` prints that the provider authentication is ready.

Common failure:

- `gh executable not found`, `glab executable not found`, `az executable not found`, or `svn executable not found`.

Next action:

- Install the provider CLI shown in the error, reopen the terminal, then rerun `gig login`.

Read-only commands do not start login for you. If `gig ABC-123` or `gig ABC-123 --repo github:owner/name` reports missing auth, run the printed `gig login <provider>` command, then retry the original command.

## 5. Inspect One Ticket

```bash
gig ABC-123
```

Use this when you want to know what changed for one ticket.

If you are outside the checkout, pass the repo target explicitly:

```bash
gig ABC-123 --repo github:owner/name
```

Supported remote targets:

- `github:owner/name`
- `gitlab:group/project`
- `bitbucket:workspace/repo`
- `azure-devops:org/project/repo`
- `svn:https://svn.example.com/repos/app/branches/staging/ProductName`

You can also pass provider URLs and common Git remotes, for example `https://github.com/owner/name`, `git@github.com:owner/name.git`, `https://dev.azure.com/org/project/_git/repo`, or a plain SVN URL. Inside the prompt, use shorter aliases such as `gh owner/name` or `repo payments`.

Expected output summary:

- ticket commits
- branches where those commits appear
- provider evidence such as PRs, merge requests, checks, deployments, issues, or work items when supported
- risk hints such as database, config, or Mendix changes

Common failure:

- no commits or evidence are found.

Next action:

- Confirm the ticket ID and current repo.
- If the ticket is only in an open PR/MR, make sure the provider has access to that repo.
- If the repo is local-only, use `gig ABC-123 --path .`.

## 6. Verify Release Readiness

```bash
gig verify ABC-123
```

Use this when you need a release decision, not just raw evidence.

Expected output summary:

- promotion path such as `staging -> main`
- verdict: `SAFE`, `WARNING`, or `BLOCKED`
- missing commits or risky evidence that explain the verdict

Common failure:

- `gig is not sure which branches represent the promotion path`

Next action:

```bash
gig verify ABC-123 --from staging --to main
```

In an interactive terminal, you can also run the short form and let `gig` ask for the missing promotion path:

```bash
gig verify ABC-123
```

If you are outside the checkout, pass the repo and let provider topology infer the path first:

```bash
gig verify ABC-123 --repo github:owner/name
```

Add `--from staging --to main` only when `gig` says the topology is ambiguous, or use an interactive terminal and answer the branch prompts.

If this is your normal project, save it:

```bash
gig
# ask gig > repo payments
# ask gig > save payments
gig verify ABC-123
```

For a shareable release-management artifact:

```bash
gig verify ABC-123 --out verify.xlsx
```

Use CSV when another spreadsheet or reporting system needs one verification
table:

```bash
gig verify ABC-123 --out verify.csv
```

## 7. Generate A Release Packet

```bash
gig packet ABC-123
```

Use this when you need a handoff artifact for QA, release review, or automation.

Expected output summary:

- ticket summary
- repository evidence
- release readiness notes
- risk and manual-review hints

Common failure:

- packet output is missing the expected branch path.

Next action:

```bash
gig packet ABC-123 --from staging --to main
```

Use JSON for tooling:

```bash
gig packet ABC-123 --json
```

Use XLSX when the packet will be shared with QA, release managers, engineering
leads, or compliance reviewers:

```bash
gig packet ABC-123 --out release-packet.xlsx
```

Packet CSV export writes a directory because a release packet has multiple
tables:

```bash
gig packet ABC-123 --format csv --out release-packet/
```

The XLSX packet includes Cover, Release Decision, Scope, Risks, Missing Changes,
Commits, Manual Steps, Verification, Approvals, Evidence, and Metadata sheets.
The CSV directory includes matching import tables such as `summary.csv`,
`release-decision.csv`, `risks.csv`, `commits.csv`, and `metadata.csv`.

Decision values are conservative: `ready` only when evidence is complete and no
blocking risk remains, `needs_review` when manual review is still required,
`blocked` when a required release check fails or changes are missing, and
`unknown` when setup or provider access prevents a reliable answer.

## 8. Save A Project When Commands Repeat

```bash
gig
# ask gig > repo payments
# ask gig > save payments

# Scriptable form:
gig project add payments --repo github:owner/name --from staging --to main --use
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

Use a saved project when the same repo and branch defaults are used repeatedly from outside a checkout. Do not create one just to try `gig`.

Common failure:

- wrong current project.

Next action:

```bash
gig project list
gig project use payments
```

The old `gig workarea ...` spelling still works.

## 9. Use Local Fallback Only When Needed

```bash
gig ABC-123 --path .
gig verify ABC-123 --path . --from staging --to main
gig packet ABC-123 --path . --from staging --to main
```

Use local mode when:

- remote provider access is unavailable
- the repo is not hosted by a supported provider
- SVN/Mendix release work depends on a local checkout

Local mode can inspect without topology. Local `verify` and `packet` need a promotion path unless a project, config, or interactive prompt supplies it.

## 10. Add Config Only After Inference Needs Help

Most teams should not start with `gig.yaml`.

Add config only when you need:

- custom environment names
- branch topology that provider metadata cannot infer
- repo owner/service metadata in output
- team-specific notes in release packets

## 11. Optional AI Assist

```bash
gig assist doctor
gig assist setup
gig explain ABC-123 --audience release-manager
gig ask "what is still blocked?"
```

Use AI assist only after the deterministic `gig` audit works. The AI layer explains `gig` evidence; it should not become the source of truth.

## Next

- [First Ticket Audit](first-ticket-audit.md)
- [Release-Day Workflow](release-day-workflow.md)
- [Troubleshooting](troubleshooting.md)
