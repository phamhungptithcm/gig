# gig

`gig` is moving toward a remote-first release audit CLI for teams that move work by ticket across many repositories.

`Did we miss any change for this ticket?`

If one ticket can touch many repos, fail review a few times, and pick up follow-up fixes before release, this is the workflow `gig` is built for.

## What You Can Do Today

With the current version of `gig`, you can:

- inspect GitHub repositories directly in supported remote flows
- inspect the full ticket story across repos
- see where a ticket is present, aligned, or behind across env branches
- verify whether the next release move looks `safe`, `warning`, or `blocked`
- capture a repeatable ticket baseline before release review
- generate a Markdown release packet for QA, client review, and release managers
- generate JSON output for CI and scripts
- save workareas and let `gig` remember inferred branch topology for repeat use
- use an optional config file only when your team needs explicit branch or repo metadata overrides
- run `gig doctor` to check inferred topology, optional overrides, and repository health

The product direction is broader than the current build:

- install and use `gig` immediately
- login to a provider once
- inspect remote branches without cloning or wiring config first
- save each project as a reusable workarea
- keep the console UX clean enough for daily multi-repo use

## Start Here

If you are new, read these in order:

1. [Product Strategy](17-product-strategy.md)
2. [Quick Start](19-quickstart.md)
3. [CLI Guide](03-cli-spec.md)
4. [Roadmap](13-roadmap.md)
5. [Config Spec](09-config-spec.md)
6. [Product Reset Audit](22-product-reset-audit.md)

## Most Useful Commands

```bash
gig login github
gig inspect ABC-123 --repo github:owner/name
gig verify --ticket ABC-123 --repo github:owner/name
gig manifest generate --ticket ABC-123 --repo github:owner/name
gig scan --path .
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
  start with the current shipped flow
- [Product Strategy](17-product-strategy.md)
  see the new product direction, workarea model, and UX priorities
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

The short version:

- today the strongest logic is ticket inspection, verification, and release packaging
- next the front door should become remote-first and zero-config
- next `gig` should remember projects as workareas
- next the console UX should become cleaner, more guided, and easier to live in
- later it can add richer evidence and carefully controlled actions
