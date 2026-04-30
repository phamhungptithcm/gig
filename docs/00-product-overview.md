# Overview

`gig` is a source-control-native release audit CLI for ticket-based delivery.

It answers:

`Did we miss any change for this ticket?`

## What gig Does

`gig` reads source-control evidence and turns it into release decisions:

- inspect one ticket across commits, branches, PRs, merge requests, deployments, checks, issues, or work items where supported
- verify whether the next promotion looks `safe`, `warning`, or `blocked`
- generate a release packet for QA, release managers, clients, or automation
- save repeated repo/branch context in projects
- keep local Git and SVN fallback available when remote access is not enough

## When To Use It

Use `gig` before a ticket or release moves forward.

It helps when:

- one ticket spans backend, frontend, database, scripts, or Mendix/low-code assets
- follow-up fixes arrived after QA, UAT, or client review
- teams need to confirm what reached the target branch
- release managers need the same evidence in terminal output, Markdown, and JSON
- manual repo-by-repo checking is too slow or inconsistent

## What gig Does Not Replace

`gig` does not replace:

- code review
- CI/CD
- issue tracking
- human release approval

It makes the release evidence easier to collect, compare, and share.

## Main Workflow

```bash
gig login
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

The same workflow works through the guided front door:

```bash
gig
```

## Product Defaults

- Current-checkout-first: inside a Git checkout, infer the remote provider target from `origin`.
- Remote-first: prefer live provider access; use `--repo github:owner/name` when outside the checkout.
- Zero-config-first: start without `gig.yaml`.
- Topology-aware: use provider branch metadata when it is confident.
- Safe fallback: if topology is ambiguous, `gig` asks for `--from`, `--to`, `--envs`, or a project instead of guessing.
- Local-optional: use `--path .` when remote access is not available.
- AI-optional: use assist commands only after deterministic `gig` evidence works.

## Best Fit Users

- developers checking whether their ticket is fully promoted
- QA/UAT coordinators validating follow-up changes
- release engineers preparing release windows
- delivery leads who need a clean audit trail

## Success State

A new user should be able to:

1. install `gig`
2. run `gig login`
3. inspect one ticket
4. verify readiness
5. export a release packet
6. add a project only when the commands become repetitive

## Next

- [Quick Start](19-quickstart.md)
- [First Ticket Audit](first-ticket-audit.md)
- [Release-Day Workflow](release-day-workflow.md)
