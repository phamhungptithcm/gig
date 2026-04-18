# gig

`gig` is a remote-first release audit CLI for one critical question:

`Did we miss any change for this ticket?`

## Start Here

Read these first if you are new:

1. [Quick Start](19-quickstart.md)
2. [Demo Guide](25-demo-guide.md)
3. [Portfolio Guide](26-portfolio-guide.md)
4. [CLI Guide](03-cli-spec.md)

## Core Workflow

- `inspect` collects the full ticket story across repositories and branches
- `verify` returns a `safe`, `warning`, or `blocked` verdict
- `manifest` exports a release packet in Markdown or JSON

AI briefings stay optional.
The source of truth stays inside `gig`.

## Fastest Path

```bash
gig
gig login github
gig ABC-123 --repo github:owner/name
gig verify ABC-123 --repo github:owner/name
gig manifest ABC-123 --repo github:owner/name
```

## Why Teams Reach For It

`gig` is a good fit when:

- one ticket can touch backend, frontend, database, scripts, or low-code assets
- follow-up fixes arrive after QA or client review
- release confidence depends on what actually reached the target branch
- teams need the same evidence in terminal output, JSON, and release packets without reopening every repo

## Documentation Map

- [Product Overview](00-product-overview.md)
  what `gig` does and where it wins
- [CLI Guide](03-cli-spec.md)
  the command surface for daily usage
- [Roadmap](13-roadmap.md)
  what is shipped, what is next, and what should not lead
- [Product Strategy](17-product-strategy.md)
  product principles and long-term direction
- [Config Spec](09-config-spec.md)
  optional overrides for mature teams
- [Product Reset Audit](22-product-reset-audit.md)
  deeper rationale behind the remote-first, zero-config direction
