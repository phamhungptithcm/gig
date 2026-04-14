# Product Overview

## Product In One Line

`gig` is a remote-first release audit CLI that reconciles ticket evidence across repositories, verifies promotion readiness, and generates release-ready output.

The memorable shorthand is:

`gig = googling in git`

## The Problem

Release teams rarely struggle to find Git history.
They struggle to answer one operational question fast enough:

`Did we miss any change for this ticket?`

That question gets harder when:

- one ticket touches several repositories
- QA or client review adds follow-up fixes
- one missed DB, config, or dependency change can block a release
- different stakeholders need the same facts in different formats

## The Promise

Before a team moves a ticket or release forward, `gig` should answer:

- what changed for this ticket
- what is still missing from the target path
- what looks risky or still needs manual review
- what evidence should be shared with QA, release, or client-facing stakeholders

## Core Workflow

- remote-first repository inspection for GitHub, GitLab, Bitbucket, Azure DevOps, and remote SVN
- `inspect` for the full ticket story
- `verify` for a `safe`, `warning`, or `blocked` release verdict
- `manifest generate` for Markdown and JSON release packets

## Why It Wins

- deterministic release reasoning instead of manual repo-by-repo checking
- zero-config-first activation instead of config-heavy onboarding
- guided terminal onboarding so first-time users can pick a repo before they learn flags
- reusable workareas for repeated project context
- optional AI explanation layered on top of auditable source-control evidence

## What Ships Today

The current build is already strong at:

- ticket inspection across repositories
- release verification and risk inference
- remote provider-backed inspection in supported live paths
- snapshot and manifest generation
- reusable workareas
- local Git and SVN fallback flows
- a guided terminal front door with arrow-key selection when users run `gig`

## Installation Note

Use the direct installer until `@hunpeolabs/gig` completes its first bootstrap publish.
If npm still returns `404`, that publish has not landed yet.

## Product Boundary

`gig` does not try to replace:

- code review systems
- CI/CD systems
- issue trackers
- human release approval on risky changes

It sits between repository history and release decisions.

## Best Fit Users

- developers and tech leads tracing the full scope of one ticket
- QA or UAT coordinators validating what changed since the previous review round
- release engineers deciding whether the next promotion is safe
- delivery leads who need a clean audit trail without manually reopening every repository

## Success State

The intended product experience is simple:

1. install `gig`
2. run `gig`
3. log in once to the right provider if needed
4. inspect a ticket or verify the next move
5. export a release packet or AI brief only when it helps
