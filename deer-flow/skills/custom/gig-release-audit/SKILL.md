---
name: gig-release-audit
description: Use this skill when the user wants a ticket or release-level audit briefing for the gig project, especially for QA, client, or release-manager audiences. Prefer gig's deterministic CLI evidence over free-form reasoning. Trigger when the user asks for release status, ticket readiness, release briefings, DeerFlow summaries for gig, or multi-ticket release review.
---

# gig Release Audit

Use `gig` as the source of truth.
Your job is to explain and prioritize deterministic release evidence, not to guess repository state.

## When To Use

Use this skill when the user asks for:

- a ticket-level release brief
- a QA, client, or release-manager summary
- a release-wide brief from many ticket snapshots or a live ticket file
- the next best `gig` command for release review

## Command Ladder

### DeerFlow Readiness

Use these before asking for an AI brief when the local sidecar may not be ready:

```bash
gig assist doctor
gig assist setup
```

### Ticket-Level Brief

Prefer:

```bash
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
```

Local fallback:

```bash
gig assist audit --ticket ABC-123 --path . --from test --to main --audience client
```

### Release-Level Brief

Only use this when snapshots already exist for the release:

```bash
gig assist release --release rel-2026-04-09 --path . --audience release-manager
```

Snapshots are expected under:

```text
.gig/releases/<release-id>/snapshots/*.json
```

If snapshots are missing, guide the workflow back to deterministic capture first:

```bash
gig snapshot create --ticket ABC-123 --from test --to main --path . --release rel-2026-04-09
```

Live remote-first mode:

```bash
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

## Audience Rules

- `qa`: focus on regression hotspots, manual checks, and risky repos first.
- `client`: focus on scope clarity, status, communication-ready risks, and what still needs confirmation.
- `release-manager`: focus on blockers, readiness, cross-ticket hotspots, and concrete next `gig` commands.

## Hard Boundaries

- Do not invent repos, commits, branches, deployments, approvals, or release outcomes.
- Preserve `safe`, `warning`, and `blocked` semantics from `gig`.
- If evidence is missing, call it out plainly.
- If the user needs raw proof, fall back to `gig inspect`, `gig verify`, `gig plan`, or `gig manifest generate`.
