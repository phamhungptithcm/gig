---
name: gig-resolve-conflict
description: Use this skill when the user wants help understanding an active Git conflict in the gig project. Prefer gig's deterministic conflict status and active-block evidence over free-form reasoning. Trigger when the user asks for DeerFlow help on gig resolve, conflict-risk explanation, scope warnings, or which resolver action looks safest.
---

# gig Resolve Conflict

Use `gig` as the source of truth.
Your job is to explain the active conflict block and safest next action, not to guess repository state or claim the conflict is already fixed.

## When To Use

Use this skill when the user asks for:

- help understanding one active merge, rebase, or cherry-pick conflict
- a release-manager, QA, or client explanation of the current conflict risk
- guidance on whether current, incoming, both, or manual edit is the safer next action
- the next best `gig resolve` command

## Command Ladder

Start with raw deterministic status:

```bash
gig resolve status --path . --ticket ABC-123
```

If the user wants a concise AI brief:

```bash
gig assist resolve --path . --ticket ABC-123 --audience release-manager
```

When they are ready to apply a choice:

```bash
gig resolve start --path . --ticket ABC-123
```

## Audience Rules

- `qa`: focus on regression risk, manual checks, and what to re-test after choosing a side.
- `client`: focus on delivery impact and what still needs confirmation before this can be described as handled.
- `release-manager`: focus on safest next action, scope warnings, and concrete next `gig` commands.

## Hard Boundaries

- Do not invent commits, branches, resolved outcomes, or Git progress that are not in the bundle.
- Do not say the conflict is resolved unless the deterministic bundle proves it.
- Do not tell the user to continue the Git operation automatically.
- Preserve `supported`, `manual`, risk, and scope-warning semantics from `gig`.
