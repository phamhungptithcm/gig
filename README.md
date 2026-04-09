# gig

`gig` helps release teams answer one question before moving code to the next environment:

`Did we miss any change for this ticket?`

If one ticket can touch many repos, fail review many times, and pick up extra fixes before release, `gig` is built for that workflow.

## What You Get Right Away

With `gig`, you can:

- find every repo touched by one ticket
- list every commit for that ticket across the workspace
- see which environments already contain that ticket and which are behind
- spot risky changes like DB, config, or Mendix-related files
- compare two branches before manual cherry-pick or promotion

Today, `gig` is read-only.
It helps you inspect and verify.
It does not change repositories by itself.
It works with Git repositories today.

## The Problem It Solves

In real teams, one ticket often touches more than one place:

- backend service
- frontend app
- database script
- low-code app such as Mendix

The same ticket can also fail several times in:

- developer verify
- QA verify
- client review
- UAT

Every failed round can add more commits.

By the time the ticket is ready for production, people usually have to answer questions like:

- Which repos changed for this ticket?
- Did we move all fixes from `dev` to `test`?
- Is `main` still missing a late follow-up commit?
- Did we forget the DB or Mendix part of the change?

That is where release mistakes happen.
`gig` helps reduce that risk.

## Who This Is For

`gig` is a good fit if:

- your team works in multiple repositories
- one ticket often has many commits over time
- you promote code through branches like `dev`, `test`, `uat`, `main`, or `prod`
- your release flow still depends on manual checking, cherry-pick, or backport work

## Will This Help My Team?

`gig` is useful when your team says things like:

- "This ticket changed three repos. Did we collect everything?"
- "QA passed, but did the last fix also reach `test`?"
- "Client asked for one more change. Did that follow-up commit reach `main` yet?"
- "This ticket has DB work too. Did we remember that part before release?"

If those questions sound familiar, this project is aimed at your workflow.

## MVP Commands Available Today

- `gig scan`
  Find repositories under a folder.
- `gig find ABC-123`
  Find commits for one ticket.
- `gig inspect ABC-123`
  Show the full ticket picture by repository, commits, branches, and risk signals.
- `gig env status ABC-123`
  Show where the ticket is present, aligned, or behind across environments.
- `gig diff --ticket ABC-123 --from dev --to test`
  Show what is still missing in the target branch.
- `gig version`
  Show the installed version.

## Quick Example

Imagine ticket `ABC-123` changed:

- `service-a`
- `web-ui`
- `billing-db`

Run:

```bash
gig inspect ABC-123 --path /path/to/workspace
gig env status ABC-123 --path /path/to/workspace --envs dev=dev,test=test,uat=uat,prod=main
gig diff --ticket ABC-123 --from test --to main --path /path/to/workspace
```

From those three commands, you can answer:

- which repos were touched
- how many commits belong to the ticket
- whether the ticket reached each environment branch
- whether a later fix is still missing from the next promotion step
- whether the ticket includes DB, config, or Mendix-style risk signals

## Install

### Option 1: Download the latest release

1. Open [GitHub Releases](https://github.com/phamhungptithcm/gig/releases/latest)
2. Download the file for your operating system
3. Extract it
4. Put `gig` or `gig.exe` somewhere on your `PATH`

This is the simplest install path for most people.

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
```

## Start In 4 Commands

### 1. Scan your workspace

```bash
gig scan --path .
```

This shows:

- which repos were found
- the repo type
- the current branch when available

### 2. Find commits for one ticket

```bash
gig find ABC-123 --path .
```

This shows:

- every repo where the ticket appears
- matching commit hashes
- commit messages
- branch information when available

### 3. Inspect the full ticket story

```bash
gig inspect ABC-123 --path .
```

This shows:

- repos touched by the ticket
- total commits per repo
- branches where those commits appear
- basic risk signals such as DB or config changes

### 4. Check environment status and what is still missing

```bash
gig env status ABC-123 --path . --envs dev=dev,test=test,prod=main
```

This shows:

- whether the ticket is present in each environment branch
- whether the next environment is behind
- which repos still need attention before promotion

You can also compare two specific branches directly:

```bash
gig diff --ticket ABC-123 --from dev --to test --path .
```

This shows:

- commits found in the source branch
- commits already present in the target branch
- commits still missing from the target branch

## Why Teams Will Care

`gig` is useful because it is:

- simple to run
- easy to read
- focused on a real release pain point
- safer than checking commit history by memory
- useful before manual cherry-pick or backport work

## What It Does Not Do Yet

Right now, `gig` does not:

- promote code automatically
- cherry-pick commits for you
- auto-resolve conflicts
- read Jira or deployment data yet
- load team config yet
- output JSON yet
- support SVN history operations yet

That is intentional.
The first goal is to make ticket discovery, environment visibility, and branch comparison reliable.

## Project Direction

`gig` is moving toward a more complete release workflow tool for teams that work across many repos and many verification rounds.

That direction includes:

- safer promotion planning
- dependency checks
- ticket snapshots
- release manifests
- Jira, deployment, and release evidence later

## Documentation

If you want more detail, start here:

- [docs/index.md](docs/index.md)
- [examples/README.md](examples/README.md)
- [docs/16-real-world-release-workflows.md](docs/16-real-world-release-workflows.md)
- [docs/17-product-strategy.md](docs/17-product-strategy.md)
- [docs/18-operating-model.md](docs/18-operating-model.md)

## In One Sentence

If your team keeps asking:

`Did we miss any change for this ticket before release?`

then `gig` is built to help answer that quickly and clearly.
