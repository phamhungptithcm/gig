# First Ticket Audit

Use this workflow when someone asks:

`What changed for ABC-123, and did all of it reach the right place?`

## 1. Start In The Repo

Remote-first:

```bash
gig ABC-123
```

If you are outside the checkout, add `--repo github:owner/name`.

Local fallback:

```bash
gig ABC-123 --path .
```

Use remote mode when you can.
Use local mode when the provider is unavailable, the repo is not supported remotely, or SVN/Mendix work depends on a checkout.

## 2. Read The Inspect Output

Look for:

- repo scope
- commit list
- branches containing the ticket
- PR/MR or deployment/check evidence when available
- risk hints such as DB, config, dependency, or Mendix changes

If `gig` returns no evidence:

- confirm the ticket format
- confirm the repo target
- check whether the ticket appears only in an open PR/MR
- run `gig login` if provider access may be missing

## 3. Verify The Promotion

```bash
gig verify ABC-123
```

If `gig` can infer the promotion path, it prints a verdict:

- `SAFE`: no missing ticket evidence was found for the path
- `WARNING`: evidence exists, but risk or manual review remains
- `BLOCKED`: expected ticket evidence is missing from the target path

If `gig` cannot infer the path:

```bash
gig verify ABC-123 --from staging --to main
```

## 4. Export The Packet

```bash
gig packet ABC-123
```

Use this when the audit needs to be shared.

For automation:

```bash
gig packet ABC-123 --json
```

## 5. Save Context If You Will Repeat This

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
```

Then the next ticket becomes:

```bash
gig ABC-456
gig verify ABC-456
gig packet ABC-456
```

## Checklist

- The repo target is correct.
- The ticket ID is correct.
- The promotion path is visible or explicitly set.
- Any `blocked` or `warning` reason is understood.
- The release packet is generated only after the evidence looks right.
