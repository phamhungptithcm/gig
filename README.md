# gig

`gig` helps teams answer one release question before moving code to the next branch or environment:

`Did we miss any change for this ticket?`

If one ticket can touch many repos, fail review many times, and pick up extra fixes before release, `gig` is built for that exact workflow.

## What Problem It Solves

In real teams, one ticket can touch:

- backend services
- frontend apps
- database scripts
- low-code apps such as Mendix

The same ticket may fail in developer verify, QA verify, client review, or UAT.
Every failed round can add more commits.

By release time, teams usually need to answer questions like:

- which repos changed for this ticket?
- is `test` still behind `dev` for this ticket?
- is `main` missing a late follow-up fix?
- does this ticket include DB or config work that needs manual review?

That is where missed commits and release mistakes happen.
`gig` helps reduce that risk.

## What You Get Today

With `gig`, you can:

- find every repo touched by one ticket
- list every commit for that ticket across a workspace
- inspect the full ticket story across repos
- check environment status such as `dev -> test -> prod`
- verify whether the next promotion step looks `safe`, `warning`, or `blocked`
- generate a read-only promotion plan in human or JSON format

Today, `gig` is read-only.
It helps you inspect, verify, and plan.
It does not cherry-pick, merge, or deploy by itself.

## Commands You Will Actually Use

- `gig --help`
  Show the command list and the main usage patterns.
- `gig inspect ABC-123 --path .`
  Show the full ticket picture across repos.
- `gig env status ABC-123 --path . --envs dev=dev,test=test,prod=main`
  Show where the ticket is present or behind in the environment flow.
- `gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main`
  Tell you if the next move looks safe.
- `gig plan --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main --format json`
  Generate a release-plan style JSON output for CI or review tooling.

## A Good First Try

```bash
gig --help
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
```

If you want the full docs site:

- GitHub Pages: [phamhungptithcm.github.io/gig](https://phamhungptithcm.github.io/gig/)
- Quick start: [Quick Start](docs/19-quickstart.md)
- CLI guide: [CLI Guide](docs/03-cli-spec.md)
- Roadmap: [Roadmap](docs/13-roadmap.md)

## Install

### Option 1: Download the latest release

1. Open [GitHub Releases](https://github.com/phamhungptithcm/gig/releases/latest)
2. Download the file for your operating system
3. Extract it
4. Put `gig` or `gig.exe` somewhere on your `PATH`

This is the simplest path for most people.

### Option 2: Build from source

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
- read Jira or deployment tools yet
- load team config from a file yet

That is intentional.
The first goal is to make release checking reliable before adding write actions.

## Project Direction

`gig` is moving toward a ticket-aware release workflow tool for multi-repo teams.

The next important steps are:

- config loading for team-specific env and branch mapping
- richer release manifests
- `doctor` checks
- Jira, PR, and deployment evidence

## In One Sentence

If your team keeps asking:

`Did we miss any change for this ticket before release?`

then `gig` is built to help answer that quickly and clearly.
