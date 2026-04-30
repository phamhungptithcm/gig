# Troubleshooting

Start here when `gig` cannot infer repo, branches, auth, or config.

## Provider Login Fails

Run:

```bash
gig login
```

If the error says a CLI is missing, install the provider CLI shown in the message:

- GitHub: `gh`
- GitLab: `glab`
- Azure DevOps: `az`
- SVN: `svn`

Then reopen the terminal and rerun `gig login`.

Read-only commands do not start interactive login. If `gig ABC-123 --repo github:owner/name` cannot access the provider, it prints the exact login command to run, for example:

```bash
gig login github
```

Then rerun the original command.

To check tools before running an audit:

```bash
gig setup --provider github
```

To let `gig` run the install command, opt in explicitly:

```bash
gig setup --provider github --install-missing
```

`gig` asks for confirmation unless `--yes` is also present.

Common install commands:

| Tool | macOS | Windows | Linux examples |
| --- | --- | --- | --- |
| `git` | `brew install git` | `winget install --id Git.Git` | `sudo apt install git`, `sudo dnf install git`, `sudo pacman -S git` |
| `gh` | `brew install gh` | `winget install --id GitHub.cli` | `sudo apt install gh`, `sudo dnf install gh`, `sudo pacman -S github-cli` |
| `glab` | `brew install glab` | `winget install --id GitLab.cli` | `sudo apt install glab`, `sudo dnf install glab`, `sudo pacman -S glab` |
| `az` | `brew install azure-cli` | `winget install --id Microsoft.AzureCLI` | `sudo apt install azure-cli`, `sudo dnf install azure-cli`, `sudo pacman -S azure-cli` |
| `svn` | `brew install subversion` | `winget install --id Apache.Subversion` | `sudo apt install subversion`, `sudo dnf install subversion`, `sudo pacman -S subversion` |

## Repo Target Is Unknown

Use a full target:

```bash
gig ABC-123 --repo github:owner/name
```

Target forms:

- `github:owner/name`
- `gitlab:group/project`
- `bitbucket:workspace/repo`
- `azure-devops:org/project/repo`
- `svn:https://svn.example.com/repos/app/branches/staging/ProductName`

## Branch Topology Is Ambiguous

`gig` should not guess when branch order is unclear.

Use explicit branches:

```bash
gig verify ABC-123 --repo github:owner/name --from staging --to main
```

If this is the normal path, save it:

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
```

For custom environment names:

```bash
gig verify ABC-123 --repo github:owner/name --envs dev=develop,test=staging,prod=main --from staging --to main
```

## Local Mode Fails With Missing Branches

Local inspect can work without topology:

```bash
gig ABC-123 --path .
```

Local release decisions need topology:

```bash
gig verify ABC-123 --path . --from staging --to main
```

## No Ticket Evidence Found

Check:

- ticket ID spelling
- repo target
- provider login
- whether the ticket only exists in an open PR/MR
- whether the ticket format matches the default pattern

Try:

```bash
gig inspect ABC-123 --repo github:owner/name
gig inspect ABC-123 --path .
```

## SVN Credentials

Do not put credentials in SVN URLs.

Use:

```bash
gig login svn
```

or:

```bash
export GIG_SVN_USERNAME=demo
export GIG_SVN_PASSWORD=secret
```

## Need Support Logs

Set:

```bash
export GIG_DIAGNOSTICS_FILE=/path/to/gig-diagnostics.jsonl
```

Then rerun the failing command.
The diagnostics file is useful for auth, API, repo, and topology issues.

## When To Add Config

Add config only after the simpler options fail.

Use config for:

- custom branch topology
- repository metadata
- team notes in output
- stable automation defaults

Do not use config as the first-run path.
