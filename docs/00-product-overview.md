# Product Overview

## What `gig` Is

`gig` is being repositioned as a remote-first release audit CLI for teams that move work by ticket across many repositories.

The core question stays the same:

`Did we miss any change for this ticket?`

That question becomes hard when one ticket keeps picking up more commits across backend, frontend, database, scripts, and low-code assets before it is finally approved.

## The Simple Promise

Before someone moves a ticket or release forward, `gig` should help them answer:

- what changed for this ticket?
- what is still missing?
- what looks risky?
- what should be reviewed by hand before approval?

## Product Direction

The intended direction is:

- login once to a remote provider and inspect live repository state
- auto-detect branch topology, ticket evidence, and likely release paths
- let users create workareas for each project so setup and inferred topology are remembered
- give `gig` with no subcommand a guided terminal front door instead of only raw help text
- keep config optional and only use it to improve quality or override detection
- make the console UX clean enough that users can stay inside `gig` while working across many repos

## Who It Should Help Most

`gig` should be strongest for:

- developers following one ticket across many repos
- QA or UAT coordinators checking what changed since the last review cycle
- release engineers who need a fast ticket audit before promotion
- delivery leads who need confidence without manually reading every repo again

## What The Current Build Already Does Well

Today, the project already has useful release logic:

- ticket inspection across repos
- branch-to-branch verification
- risk hints for DB, config, and Mendix-style changes
- Markdown and JSON release packets
- GitHub, GitLab, Bitbucket, and Azure DevOps-backed remote inspection for selected flows
- remote SVN inspection with login-backed credentials
- initial workarea save and switch flow for project defaults plus remembered inferred branch topology
- a guided root dashboard when users run `gig` with no subcommand
- a local `gig assist doctor` readiness check for the bundled DeerFlow sidecar
- experimental DeerFlow-backed ticket, release, and conflict briefings built from `gig` evidence bundles
- a local `gig assist setup` bootstrap for the bundled DeerFlow sidecar
- local workspace scanning and optional config overrides

## Installation

The public npm package is `@phamhungptithcm/gig`, but the command you run after install stays `gig`.

Install with npm:

```bash
npm install -g @phamhungptithcm/gig
gig version
```

Upgrade an npm install:

```bash
gig update
gig update v2026.04.09
```

If you prefer raw npm commands:

```bash
npm install -g @phamhungptithcm/gig@latest
npm install -g @phamhungptithcm/gig@2026.4.9
```

For macOS and Linux without npm:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
gig version
```

Update a direct install:

```bash
gig update
```

## What It Does Not Do Yet

Today, `gig` does not:

- give a zero-config first-run experience across all providers
- provide the full keyboard-first console browser and drill-down UX the product needs yet
- gather issue-tracker evidence yet
- move commits or deploy anything for you

The current AI briefing slice is optional and additive.
It now supports audience-specific ticket briefs, release-level briefs from saved snapshots or live ticket sets, and local conflict-resolution briefs for active Git conflicts.
Core release answers still come from deterministic `gig` services, not from an LLM guessing repository state.

Those are product-direction gaps, not a change in purpose.
The release-audit problem is still the center.

## Why This Matters

The problem is not only "find commits by ticket."

The real problem is:

- a ticket may fail review many times
- different repos may move at different speeds
- release managers can easily miss one late follow-up commit
- DB, config, or Mendix changes may need manual review

`gig` is meant to reduce those mistakes.

## What Success Should Look Like

After install, users should be able to:

1. run `gig`
2. log in once if needed
3. inspect a ticket online across remote branches
4. let `gig` remember that project automatically or switch to a saved workarea later
5. understand risk and missing changes without wiring config first
