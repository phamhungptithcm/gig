# gig

`gig` helps teams answer one important release question before moving code forward:

`Do we really have every change for this ticket?`

If your team works across many repos and one ticket can be fixed many times before it is finally approved, `gig` is built for that exact problem.

## What You Get

With `gig`, you can:

- find every repo touched by one ticket
- find every commit for that ticket across the workspace
- compare two branches and see what is still missing
- check release readiness before manual cherry-pick or promotion

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

The same ticket may also fail several times in:

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
- you want a safer release check before manual promotion

## What `gig` Can Do Today

Current commands:

- `gig scan`
- `gig find`
- `gig diff`
- `gig version`

What each one does:

- `gig scan` finds repositories under a folder
- `gig find ABC-123` finds commits for one ticket
- `gig diff --ticket ABC-123 --from dev --to test` shows what is still missing in the target branch
- `gig version` shows the installed version

## Quick Example

Imagine ticket `ABC-123` changed:

- `service-a`
- `web-ui`
- `billing-db`

Run:

```bash
gig diff --ticket ABC-123 --from test --to main --path /path/to/workspace
```

`gig` will group results by repository and show where `main` is still missing ticket commits.

That is the key check before manual promotion.

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

## Start In 3 Commands

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

### 3. Check what is missing in the next branch

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
- load team config yet
- output JSON yet
- support SVN history operations yet

That is intentional.
The first goal is to make ticket discovery and branch comparison reliable.

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
