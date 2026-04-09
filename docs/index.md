# gig

`gig` is a small CLI that helps teams answer one release question:

`Did we miss any change for this ticket before moving it forward?`

If your team works across many repos, verifies the same ticket many times, and still promotes changes by branch, `gig` is built for that problem.

## What You Can Do Today

With the current version of `gig`, you can:

- scan a workspace and find repos quickly
- find every commit for one ticket
- inspect the full ticket story across repos
- check whether a ticket is behind in `dev`, `test`, or `main`
- verify whether the next promotion step looks safe
- generate a read-only promotion plan in human or JSON format

Everything is read-only right now.
`gig` helps you inspect, verify, and plan.
It does not cherry-pick, merge, or deploy by itself.

## Start Here

If you are new, read these in order:

1. [Quick Start](19-quickstart.md)
2. [CLI Guide](03-cli-spec.md)
3. [Roadmap](13-roadmap.md)

## Most Useful Commands

```bash
gig --help
gig inspect ABC-123 --path .
gig verify --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main
gig plan --ticket ABC-123 --from test --to main --path . --envs dev=dev,test=test,prod=main --format json
```

## When `gig` Is A Good Fit

`gig` is useful when your team says things like:

- "This ticket touched three repos. Did we collect everything?"
- "QA passed, but is `test` still behind `dev` for this ticket?"
- "Client asked for one more fix. Did that commit reach `main` yet?"
- "This ticket includes DB work. Do we need a manual review before release?"

## What The Docs Cover

- [Product Overview](00-product-overview.md)
  explains what the project is and who it helps
- [CLI Guide](03-cli-spec.md)
  shows every command and when to use it
- [Branching And Release](15-branching-and-release.md)
  explains this repo's own release flow
- [Real-World Release Workflows](16-real-world-release-workflows.md)
  shows the real team situations the tool is designed for
- [Roadmap](13-roadmap.md)
  shows what is already here and what comes next

## Current Direction

`gig` is moving toward a ticket-aware release workflow tool for multi-repo teams.

The short version:

- today it helps you see and verify
- next it should help you produce better release packets and team config
- later it can support richer release evidence and controlled execution
