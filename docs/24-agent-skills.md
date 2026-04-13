# Agent Skills

This page summarizes the project-specific skills added for AI agents that work with `gig`.

The canonical repo-level index is still `SKILLS.md` in the project root.
This docs page exists so the published docs can point to the same guidance without breaking the MkDocs build.

## Why These Skills Exist

The main risk with AI inside `gig` is not lack of intelligence.
It is drifting away from the deterministic release engine.

These skills keep agents aligned with the real product shape:

- `gig` computes ticket, branch, and release facts
- AI explains, prioritizes, and helps choose next actions
- prompts and memory do not replace the source of truth

## Included Skills

### `gig-release-audit`

Location:

```text
deer-flow/skills/custom/gig-release-audit/SKILL.md
```

Use when an agent needs to:

- brief one ticket for `qa`, `client`, or `release-manager`
- brief a named release from many saved ticket snapshots or a live ticket file
- pick the correct `gig` command for release review

Main commands:

```bash
gig assist audit --ticket ABC-123 --repo github:owner/name --audience qa
gig assist audit --ticket ABC-123 --path . --from test --to main --audience client
gig assist release --release rel-2026-04-09 --path . --audience release-manager
gig assist release --release rel-2026-04-09 --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

### `gig-resolve-conflict`

Location:

```text
deer-flow/skills/custom/gig-resolve-conflict/SKILL.md
```

Use when an agent needs to:

- explain one active Git conflict in `gig`
- call out scope warnings and risk before the user chooses a resolver action
- recommend the next `gig resolve` command without claiming the conflict is already fixed

Main commands:

```bash
gig resolve status --path . --ticket ABC-123
gig assist resolve --path . --ticket ABC-123 --audience release-manager
gig resolve start --path . --ticket ABC-123
```

### `gig-product-guardrails`

Location:

```text
deer-flow/skills/custom/gig-product-guardrails/SKILL.md
```

Use when an agent is changing:

- CLI behavior
- provider-backed flows
- user-facing docs
- AI integration boundaries

This skill reinforces:

- remote-first, source-control-native behavior
- zero-config-first onboarding
- thin CLI, service-heavy architecture
- docs and tests staying in sync with user-facing behavior

## Root Index

For the full repo-facing version, see:

```text
SKILLS.md
```
