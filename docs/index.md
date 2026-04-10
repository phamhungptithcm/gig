# gig

`gig` helps teams answer one release question before they move code forward:

`Did we miss any change for this ticket?`

If one ticket can touch many repos, fail review a few times, and pick up follow-up fixes before release, this is the workflow `gig` is built for.

## What You Can Do Today

With the current version of `gig`, you can:

- scan a workspace and find repos quickly
- inspect the full ticket story across repos
- see where a ticket is present, aligned, or behind across env branches
- verify whether the next release move looks `safe`, `warning`, or `blocked`
- capture a repeatable ticket baseline before release review
- generate a Markdown release packet for QA, client review, and release managers
- generate JSON output for CI and scripts
- use a config file for real branch names, repo owners, service names, and repo types
- run `gig doctor` to check config coverage and environment mapping health

Everything is read-only right now.
`gig` helps people inspect, verify, and prepare before a risky release move happens.

## Start Here

If you are new, read these in order:

1. [Quick Start](19-quickstart.md)
2. [CLI Guide](03-cli-spec.md)
3. [Config Spec](09-config-spec.md)
4. [Roadmap](13-roadmap.md)
5. [Dependency Slice Plan](20-dependency-risk-slice.md)
6. [Product Reset Audit](22-product-reset-audit.md)

## Most Useful Commands

```bash
gig --help
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path .
gig snapshot create --ticket ABC-123 --from test --to main --path . --output .gig/snapshots/abc-123.json
gig snapshot create --ticket ABC-123 --from test --to main --path . --release rel-2026-04-09
gig plan --release rel-2026-04-09 --path .
gig verify --release rel-2026-04-09 --path .
gig manifest generate --release rel-2026-04-09 --path .
gig manifest generate --ticket ABC-123 --from test --to main --path .
gig doctor --path .
```

## When `gig` Is A Good Fit

`gig` is useful when your team says things like:

- "This ticket touched three repos. Did we collect everything?"
- "QA passed, but is `test` still behind `dev` for this ticket?"
- "Client asked for one more fix. Did that last commit reach `main` yet?"
- "This ticket includes DB or config work. What needs manual review before release?"

## What The Docs Cover

- [Quick Start](19-quickstart.md)
  start fast with the commands most teams use first
- [CLI Guide](03-cli-spec.md)
  see every command, when to use it, and sample output formats
- [Config Spec](09-config-spec.md)
  map real branches, services, owners, and repo kinds
- [Branching And Release](15-branching-and-release.md)
  understand this repo's own release flow
- [Roadmap](13-roadmap.md)
  see what is shipped, what is next, and what is later
- [Dependency Slice Plan](20-dependency-risk-slice.md)
  see the concrete implementation breakdown for dependency parsing and missing-dependency risk
- [Product Reset Audit](22-product-reset-audit.md)
  evaluate the current product shape and the recommended remote-first reset

## Current Direction

`gig` is moving toward a ticket-aware release workflow tool for multi-repo teams.

The short version:

- today it helps you see, verify, and package release information
- today it can also save ticket snapshots for audit and re-check
- next it should add richer evidence from Jira, PRs, and deployments
- later it can support release bundles and controlled execution
