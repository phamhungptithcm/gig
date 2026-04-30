# Conflict Resolution

Conflict commands are optional.
Use them only after Git has already stopped on a conflict.

## Inspect Conflict State

```bash
gig resolve status --path .
```

This summarizes the active conflict and ticket context when available.

## Start Assisted Resolution

```bash
gig resolve start --path . --ticket ABC-123
```

`gig resolve start` can help with supported text conflicts.
It does not continue the Git operation or create a commit for you.

## Optional AI Brief

```bash
gig assist resolve --path . --ticket ABC-123 --audience release-manager
```

Use this when you want a human-readable explanation of the deterministic conflict bundle.

## Boundaries

- Use Git status and tests before committing.
- Do not treat conflict assist as automatic merge approval.
- Keep release decisions grounded in `gig verify` after the conflict is resolved.
