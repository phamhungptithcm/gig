# gig

`gig` is being reshaped into a remote-first release audit CLI for teams that move work by ticket across many repositories.

`Did we miss any change for this ticket?`

One ticket can touch backend, frontend, database, scripts, low-code assets, and late follow-up fixes across several repos. `gig` should collect that evidence directly from the user's source-control account, reduce setup, and give a clear next release decision fast.

## Product Direction

`gig` should feel simple after install:

- install it and run it immediately
- sign in once to GitHub, GitLab, Bitbucket, or SVN when needed
- search ticket evidence online across remote branches by default
- auto-detect protected branches, likely release flow, and repo relationships
- remember each product or client setup as a workarea so the user can come back later and continue
- keep config as an optional upgrade, not a first-run requirement
- present a console experience that is calm, keyboard-friendly, readable, and easy to drill into

## The Real Pain Points

`gig` should solve these problems first:

- "I do not know which repos this ticket touched."
- "This ticket failed QA twice and picked up follow-up fixes. Did we catch all of them?"
- "I should not have to wire config before the tool becomes useful."
- "I work across many projects and repos. I want the tool to remember each setup."
- "I need terminal output that is readable for humans and structured enough for audit."

## North-Star Experience

The intended front door is:

1. install `gig`
2. run `gig`
3. sign in to a provider if needed
4. choose or create a project workarea
5. inspect a ticket or release
6. get a clear audit result with evidence, risk, and next actions

That is the direction.
The current build already has useful release logic, but it still leans more on local workspace scanning and manual config than the target product should.

## What You Can Do With It Today

The current build already does these parts well:

- inspect the full ticket story across repositories
- verify whether a promotion looks `safe`, `warning`, or `blocked`
- generate Markdown and JSON release packets
- inspect GitHub repositories directly with login-backed remote access
- scan a local workspace when remote access is not enough
- load team config and branch overrides when a mature team needs more control
- inspect and walk supported Git text conflicts from the terminal

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

If you want the best current workflow today, start with a remote repository first:

```bash
gig login github
gig inspect ABC-123 --repo github:owner/name
gig verify --ticket ABC-123 --repo github:owner/name
gig manifest generate --ticket ABC-123 --repo github:owner/name
```

If your team needs local fallback or explicit overrides, add:

```bash
gig scan --path .
gig doctor --path .
gig resolve status --path .
```

If you want a team-specific setup, create a `gig.yaml` file and then run the same commands without repeating `--envs` every time.
If you are working directly against GitHub, `gig` can authenticate through `gh` and read repository state live without cloning first.

## A Friendly First Workflow

### 1. Connect to GitHub once

```bash
gig login github
```

### 2. Inspect one ticket directly on GitHub

```bash
gig inspect ABC-123 --repo github:owner/name
```

### 3. Check whether the next move looks safe

```bash
gig verify --ticket ABC-123 --repo github:owner/name
```

`gig` will try to infer the protected-branch release path automatically for remote GitHub repositories.

### 4. Generate a release packet people can actually read

```bash
gig manifest generate --ticket ABC-123 --repo github:owner/name
```

### 5. If needed, fall back to a local workspace

### 5.1 See what repos are under the workspace

```bash
gig scan --path .
```

### 5.2 Inspect one ticket across repos

```bash
gig inspect ABC-123 --path .
```

### 5.3 Check whether the next move looks safe

```bash
gig verify --ticket ABC-123 --from test --to main --path .
```

### 5.4 Generate a release packet people can actually read

```bash
gig manifest generate --ticket ABC-123 --from test --to main --path .
```

### 5.5 Check whether your config and repo mapping are healthy

```bash
gig doctor --path .
```

### 5.6 If Git stops on conflicts, inspect or resolve them

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
- Product strategy: [docs/17-product-strategy.md](docs/17-product-strategy.md)
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
