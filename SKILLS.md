# SKILLS

This file defines the project-specific skills and guardrails that AI agents should use when working in `gig`.

`gig` is a deterministic release-audit CLI.
Agents should use AI to explain, prioritize, and orchestrate, but not to replace the source of truth.

## Core Rules

- Treat `gig` output as the source of truth for ticket scope, branch comparisons, risk signals, verification verdicts, and release readiness.
- Prefer remote-first and zero-config-first flows when the product can answer from source-control metadata.
- Keep the CLI thin. Put product behavior in `internal/*` services and provider behavior in `internal/scm/*`.
- If the bundle says `warning` or `blocked`, preserve that severity in any AI summary.
- Do not invent commits, branches, approvals, deployments, or release outcomes that are not present in `gig` evidence.

## Available Project Skills

### `gig-release-audit`

Use when the user wants:

- a ticket-level AI brief
- a QA, client, or release-manager summary
- a release-level brief from many saved snapshots or a live ticket file
- help deciding which `gig` command to run next

Main commands:

```bash
gig assist doctor
gig assist setup
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
gig assist audit --ticket ABC-123 --path . --from test --to main --audience client
gig assist release --release rel-2026-04-09 --path . --audience release-manager
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

Expected behavior:

- Use `gig assist doctor` when the user needs to know whether the bundled DeerFlow sidecar is configured, startable, and reachable.
- Use `gig assist setup` when the local sidecar still needs bootstrap files.
- Use `gig assist audit` for one ticket.
- Use `gig assist release` with `--path` only when the release already has saved ticket snapshots under `.gig/releases/<release-id>/snapshots/`.
- Use `gig assist release` with `--ticket` or `--ticket-file` when the release bundle should be built live from local or remote evidence.
- Fall back to `gig inspect`, `gig verify`, `gig plan`, and `gig manifest generate` when the user needs raw evidence instead of AI narration.

### `gig-resolve-conflict`

Use when the user wants:

- help understanding one active Git conflict inside `gig`
- a QA, client, or release-manager explanation of the active conflict block
- help deciding whether to accept current, incoming, both, or edit manually

Main commands:

```bash
gig resolve status --path . --ticket ABC-123
gig assist resolve --path . --ticket ABC-123 --audience release-manager
gig resolve start --path . --ticket ABC-123
```

Expected behavior:

- Use `gig assist resolve` to explain the current active conflict block, not to resolve it automatically.
- Preserve scope warnings, risk notes, and supported/manual file distinctions from `gig`.
- Fall back to `gig resolve status` when the user needs raw conflict facts.
- Fall back to `gig resolve start` when the user is ready to apply a choice in the working tree.

### `gig-product-guardrails`

Use when changing CLI behavior, provider flows, or user-facing docs.

Expected behavior:

- Prefer source-control-native access over local-only assumptions.
- Reduce required flags when protected-branch or topology inference can answer.
- Update `README.md`, `docs/03-cli-spec.md`, `docs/19-quickstart.md`, `docs/00-product-overview.md`, `docs/13-roadmap.md`, and `docs/22-product-reset-audit.md` when user-facing behavior changes.
- Run focused tests first, then broader verification when the slice is stable.

## DeerFlow Skill Paths

These project skills are also available as DeerFlow custom skills:

- `deer-flow/skills/custom/gig-release-audit/SKILL.md`
- `deer-flow/skills/custom/gig-resolve-conflict/SKILL.md`
- `deer-flow/skills/custom/gig-product-guardrails/SKILL.md`
