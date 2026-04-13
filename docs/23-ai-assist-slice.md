# AI Assist Slice

## Purpose

This page captures the first practical AI integration slice for `gig`.

The goal is not to turn `gig` into a generic chat CLI.
The goal is to let `gig` keep owning deterministic release evidence while an AI layer turns that evidence into a more executive, more readable briefing.

## Core Rule

`gig` remains the source of truth.

That means:

- `gig` computes ticket scope, branch comparisons, risk signals, dependency status, and verification verdicts
- AI consumes the resulting bundle
- AI explains and prioritizes
- AI does not invent missing branch or commit facts

## Why DeerFlow Fits

DeerFlow already has the pieces that are useful on top of `gig`:

- a lead agent with tool orchestration
- optional sub-agents for deeper follow-up work
- memory and skill support for reusable workflows
- a simple HTTP surface for thread creation and streaming responses

That makes it a good sidecar for:

- executive summaries
- release-manager briefings
- evidence-gap callouts
- recommended next commands
- future multi-source evidence synthesis across SCM, CI, deployments, and issue trackers

## Boundary

The integration boundary should stay clean:

- `internal/inspect`, `internal/plan`, and `internal/manifest` keep their current deterministic responsibilities
- `internal/assistant` builds an audit bundle from those services
- the DeerFlow client receives that bundle and asks for a briefing
- CLI output renders the final AI summary, but the structured `gig` result still exists underneath

This keeps the CLI thin and avoids pushing model-specific logic into the release engine.

## Current Slices

The current shipped commands are:

```bash
gig assist doctor
gig assist setup
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
gig assist release --release rel-2026-04-09 --path . --audience release-manager
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
gig assist resolve --path . --ticket ABC-123 --audience release-manager
```

`gig assist doctor`:

1. finds the bundled `deer-flow/` workspace inside the `gig` repo
2. checks whether the local sidecar is configured, startable, and reachable
3. reports the next onboarding step without writing files

`gig assist setup`:

1. finds the bundled `deer-flow/` workspace inside the `gig` repo
2. creates `config.yaml` and any available DeerFlow env templates when needed
3. tells the user the recommended next start command for the local sidecar

`gig assist audit`:

1. resolves repo scope and branch context using normal `gig` rules
2. builds a deterministic bundle from inspection, planning, verification, and manifest highlights
3. sends that bundle to DeerFlow
4. prints a concise AI briefing tuned for `qa`, `client`, or `release-manager`

`gig assist release`:

1. loads saved ticket snapshots from one named release, or captures a live ticket set from local or remote repositories
2. builds a deterministic release bundle with release-plan rollups and packet data
3. sends that bundle to DeerFlow
4. prints a concise AI release briefing for the selected audience

`gig assist resolve`:

1. loads the deterministic `gig resolve` status plus the first active supported conflict block
2. builds a conflict bundle with provenance, risk, scope warnings, and supported resolver actions
3. sends that bundle to DeerFlow
4. prints a concise AI conflict briefing for the selected audience

These commands are intentionally experimental.
It is an additive explanation layer, not a required step in the main product flow.

## What This Slice Solves

This slice helps with the practical pain points that deterministic JSON alone does not solve well:

- stakeholders want a short summary, not raw repo data
- release managers want obvious next commands
- teams want risk phrased in business-readable language
- people want the same facts explained differently for QA, client review, and release handoff
- release owners want one release-wide summary across many ticket snapshots or live ticket sets instead of reading each ticket separately
- developers want help understanding which conflict choice is safest without replacing the deterministic resolver

## What This Slice Does Not Do

It does not:

- replace `verify`
- replace `manifest generate`
- guess release state without `gig` evidence
- require DeerFlow for the main remote audit workflow
- change `gig` into a write-enabled automation tool
- replace `gig resolve start` as the actual place where conflict choices are applied

## Skill Support

The repo now includes project-specific skill docs so DeerFlow or other agents can stay aligned with `gig`:

- [Agent Skills](24-agent-skills.md)
- `deer-flow/skills/custom/gig-release-audit/SKILL.md`
- `deer-flow/skills/custom/gig-resolve-conflict/SKILL.md`
- `deer-flow/skills/custom/gig-product-guardrails/SKILL.md`

These skills reinforce the main boundary:

- use `gig` as the evidence engine
- use AI to explain and prioritize
- do not let prompts replace branch, ticket, or risk reasoning
- support both saved-snapshot and live remote release bundle workflows

## Near-Term Direction

After the current slices, the next high-value steps are:

- expand the release-level bundle with richer cross-ticket evidence such as PR, CI, deployment, and issue-tracker context
- make repeated AI briefs work more naturally through saved workareas so users need fewer scope flags
- expose `gig` itself as a tool or MCP surface so DeerFlow can request fresh evidence instead of relying only on prompt text
- add richer prompt templates inside the project skill pack without moving logic into prompts

## Product Positioning

The product story becomes stronger with this framing:

`gig is a deterministic release-audit engine with an optional AI briefing layer.`

That is more professional than a generic AI assistant and more useful than a plain branch-comparison tool.
