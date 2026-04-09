# gig

`gig` helps teams answer one release question before they move code to the next branch or environment:

`Did we miss any change for this ticket?`

This project is for teams where one ticket can touch many repos, fail review a few times, get follow-up fixes, and then become hard to release safely.

## What You Can Do With It Today

With `gig`, you can:

- find every repo touched by one ticket
- inspect the full ticket story across repos
- see whether `test` is behind `dev`, or `main` is behind `test`
- get a quick `safe`, `warning`, or `blocked` decision before promotion
- generate a Markdown release packet for QA, client review, and release managers
- generate JSON output for CI, scripts, and tooling
- load a simple team config file so real branch names and repo ownership match your workflow
- run `gig doctor` to check whether the config and repo mapping are good enough to trust
- inspect an active Git conflict state and walk supported text conflicts from the terminal

Most commands are still read-only.
`gig` helps you inspect, verify, and prepare.
It now also helps with active Git conflict resolution, but it still does not cherry-pick, merge, or deploy for you automatically.

## Who This Is For

`gig` is a good fit when your team says things like:

- "This ticket touched backend, frontend, and DB. Did we collect everything?"
- "QA passed, but is `test` still missing one late fix from `dev`?"
- "Client failed review once, then we fixed it again. Did that last commit reach `main`?"
- "This ticket has DB or config changes. What needs manual review before release?"

## The Commands People Usually Start With

```bash
gig --help
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path .
gig manifest generate --ticket ABC-123 --from test --to main --path .
gig doctor --path .
gig resolve status --path .
```

If you want a team-specific setup, create a `gig.yaml` file and then run the same commands without repeating `--envs` every time.

## A Friendly First Workflow

### 1. See what repos are under the workspace

```bash
gig scan --path .
```

### 2. Inspect one ticket across repos

```bash
gig inspect ABC-123 --path .
```

### 3. Check whether the next move looks safe

```bash
gig verify --ticket ABC-123 --from test --to main --path .
```

### 4. Generate a release packet people can actually read

```bash
gig manifest generate --ticket ABC-123 --from test --to main --path .
```

### 5. Check whether your config and repo mapping are healthy

```bash
gig doctor --path .
```

### 6. If Git stops on conflicts, inspect or resolve them

```bash
gig resolve status --path .
gig resolve start --path .
```

## Team Config In One Minute

If your branches are not just `dev`, `test`, and `main`, add a config file like this:

```yaml
ticketPattern: '\b[A-Z][A-Z0-9]+-\d+\b'

environments:
  - name: dev
    branch: develop
  - name: test
    branch: release/test
  - name: prod
    branch: main

repositories:
  - path: services/accounts-api
    service: Accounts API
    owner: Backend Team
    kind: app
    notes:
      - Verify login and billing summary in QA.
```

Supported file names:

- `gig.yaml`
- `gig.yml`
- `.gig.yaml`
- `.gig.yml`

`gig` will auto-detect the file from the path you run against, or you can pass `--config`.

There is also a ready sample here:

- [gig.example.yaml](https://github.com/phamhungptithcm/gig/blob/main/gig.example.yaml)

## Docs

- GitHub Pages: [phamhungptithcm.github.io/gig](https://phamhungptithcm.github.io/gig/)
- Quick start: [docs/19-quickstart.md](docs/19-quickstart.md)
- CLI guide: [docs/03-cli-spec.md](docs/03-cli-spec.md)
- Config spec: [docs/09-config-spec.md](docs/09-config-spec.md)
- Roadmap: [docs/13-roadmap.md](docs/13-roadmap.md)

## Install

Package managers publish `gig` as `gig-cli` to avoid name collisions, but the command you run after install stays `gig`.
Homebrew and Scoop track the latest stable release. If you want to pin a specific version, use the direct installer or download the release asset manually.

### Quick Install On macOS And Linux With Homebrew

```bash
brew tap phamhungptithcm/gig https://github.com/phamhungptithcm/gig
brew install phamhungptithcm/gig/gig-cli
gig version
```

Update to the latest Homebrew release:

```bash
brew update
brew upgrade gig-cli
```

### Quick Install On Windows With Scoop

```powershell
scoop bucket add gig https://github.com/phamhungptithcm/gig
scoop install gig/gig-cli
gig version
```

Update to the latest Scoop release:

```powershell
scoop update gig-cli
```

### Quick Install On macOS And Linux Without Homebrew

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh -s -- --version v2026.04.09
```

Update a direct install:

```bash
gig update
gig update v2026.04.09
```

### Quick Install On Windows PowerShell

Install the latest release:

```powershell
irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1 | iex
```

Install a specific version:

```powershell
& ([ScriptBlock]::Create((irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1))) -Version v2026.04.09
```

Update a direct install:

```powershell
gig update
gig update v2026.04.09
```

### Manual Download

1. Open [GitHub Releases](https://github.com/phamhungptithcm/gig/releases/latest)
2. Download the file for your operating system
3. Extract it
4. Put `gig` or `gig.exe` somewhere on your `PATH`

### Build From Source

Requirements:

- Go `1.23+`
- Git installed and available on your `PATH`

Build:

```bash
git clone https://github.com/phamhungptithcm/gig.git
cd gig
mkdir -p bin
go build -o bin/gig ./cmd/gig
```

Run:

```bash
./bin/gig version
./bin/gig --help
```

## What `gig` Does Not Do Yet

Right now, `gig` does not:

- move code automatically
- read Jira, PR, or deployment evidence yet
- build multi-ticket release bundles yet

Those are still on the roadmap.
The current focus is to make ticket-based release checking useful, reliable, and easy to adopt first.
