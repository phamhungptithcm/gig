# gig

`gig` is a cross-platform Go CLI for release workflows that need to track ticket-related commits across multiple repositories before promotion.

## Why It Exists

Enterprise tickets often span multiple repositories and multiple rounds of QA and client review. During promotion, teams can easily miss follow-up commits, especially when the same ticket is fixed repeatedly across `dev`, `test`, and later branches. `gig` reduces that risk by scanning a workspace, finding commits by ticket, and surfacing branch gaps before any promotion step is attempted.

## Current MVP Scope

The current MVP focuses on safe, read-only workflows:

- recursive workspace scanning
- Git repository detection
- ticket-based commit search across repositories
- branch comparison for a ticket using Git-first logic
- human-readable grouped CLI output

SVN is intentionally left as a prepared adapter stub for future phases.

## Project Layout

- `cmd/gig`: executable entrypoint
- `internal/cli`: command parsing and command orchestration
- `internal/repo`: workspace and repository discovery
- `internal/scm`: shared SCM abstractions and adapter registry
- `internal/scm/git`: Git adapter implementation
- `internal/scm/svn`: SVN stub adapter for future phases
- `internal/ticket`: ticket parsing and commit search service
- `internal/diff`: branch comparison service
- `internal/output`: human-readable rendering
- `internal/config`: default configuration values
- `docs/`: product and architecture documentation
- `examples/`: sample usage

## Requirements

- Go 1.22+
- Git CLI available on `PATH` for Git-backed commands

## Quick Install

### macOS With Homebrew

```bash
brew install https://raw.githubusercontent.com/phamhungptithcm/gig/main/Formula/gig.rb
```

### Windows With Scoop

```powershell
scoop install https://raw.githubusercontent.com/phamhungptithcm/gig/main/Scoop/gig.json
```

### macOS And Linux Script Installer

Install the latest release with one command:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | GIG_VERSION=v0.1.0 sh
```

### Windows PowerShell Installer

Install the latest release from PowerShell:

```powershell
irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1 | iex
```

Install a specific version:

```powershell
$env:GIG_VERSION="v0.1.0"; irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1 | iex
```

### Manual Download

If you prefer not to use installer scripts, download the right archive from [GitHub Releases](https://github.com/phamhungptithcm/gig/releases/latest), extract it, and put `gig` or `gig.exe` on your `PATH`.

After install, verify it:

```bash
gig version
```

If you are preparing the very first public release, package manager installs become available after that first GitHub Release is published.

## Build And Run

Build the CLI:

```bash
mkdir -p bin && go build -o bin/gig ./cmd/gig
```

Run directly with Go:

```bash
go run ./cmd/gig scan --path .
```

## Commands

Scan a workspace or a single repository:

```bash
gig scan --path .
```

Find commits by ticket across detected repositories:

```bash
gig find ABC-123 --path .
```

Compare ticket-related commits between branches:

```bash
gig diff --ticket ABC-123 --from dev --to test --path .
```

## Output Behavior

- results are grouped by repository
- errors go to stderr
- invalid usage returns a non-zero exit code
- MVP commands are non-destructive

## Limitations

- Git is the only working SCM adapter in the MVP
- commit matching depends on ticket IDs being present in commit messages
- no JSON output yet
- no promote/cherry-pick workflow yet
- no config file loading yet

## Development Flow

- `staging` is the shared integration branch.
- feature and bug-fix branches start from `staging` and open pull requests back into `staging`
- `main` receives the scheduled promotion from `staging`

## Repository Automation

- CI runs formatting, vet, test, and build checks for pushes and pull requests on `staging` and `main`
- every push to `main` creates the next GitHub release tag, release notes, checksums, and release archives
- Markdown documentation under `docs/` is published to GitHub Pages with MkDocs

## Roadmap Summary

- Phase 0: foundation, CLI bootstrap, repo discovery
- Phase 1: `scan`, `find`, `diff`
- Phase 2: promotion planning and dry-run cherry-pick
- Phase 3+: dependency parsing, snapshots, SVN, Jira, Mendix checks

See [examples/README.md](examples/README.md) and the documents in [docs/](docs/) for more detail.
