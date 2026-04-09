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

Everything is still read-only.
`gig` helps you inspect, verify, and prepare.
It does not cherry-pick, merge, or deploy for you.

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

### Quick Install On macOS With Homebrew

```bash
brew install https://raw.githubusercontent.com/phamhungptithcm/gig/main/Formula/gig.rb
```

### Quick Install On Windows With Scoop

```powershell
scoop install https://raw.githubusercontent.com/phamhungptithcm/gig/main/Scoop/gig.json
```

### Quick Install On macOS And Linux

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | GIG_VERSION=v0.1.0 sh
```

### Quick Install On Windows PowerShell

Install the latest release:

```powershell
irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1 | iex
```

Install a specific version:

```powershell
$env:GIG_VERSION="v0.1.0"; irm https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.ps1 | iex
```

### Manual Download

1. Open [GitHub Releases](https://github.com/phamhungptithcm/gig/releases/latest)
2. Download the file for your operating system
3. Extract it
4. Put `gig` or `gig.exe` somewhere on your `PATH`

### Build From Source

Requirements:

- Go `1.22+`
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
- resolve merge conflicts
- read Jira, PR, or deployment evidence yet
- build multi-ticket release bundles yet

Those are still on the roadmap.
The current focus is to make ticket-based release checking useful, reliable, and easy to adopt first.
