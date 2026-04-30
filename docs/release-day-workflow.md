# Release-Day Workflow

Use this workflow when the question changes from one ticket to a release batch:

`Is this release ready to move?`

## Recommended Setup

Save a project once:

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
```

Then reuse it during the release window:

```bash
gig project use payments
```

## Verify A Release

```bash
gig verify --release rel-2026-04-09
```

Use this when the release scope is already known to `gig`.

If you are using a ticket list:

```bash
gig verify --ticket-file tickets.txt --repo github:owner/name
```

Expected output:

- release verdict
- tickets or repositories that are blocked
- manual-review risks
- missing commits or dependency evidence

## Export The Release Packet

```bash
gig packet --release rel-2026-04-09
```

Or with a ticket file:

```bash
gig packet --ticket-file tickets.txt --repo github:owner/name
```

Use Markdown for humans and JSON for automation:

```bash
gig packet --release rel-2026-04-09 --format json
```

## Optional AI Brief

```bash
gig assist release --release rel-2026-04-09 --audience release-manager
```

Use this only after `verify` and `packet` work.
The AI brief explains the deterministic release bundle; it does not replace it.

## Common Release-Day Failures

| Failure | Meaning | Next action |
| --- | --- | --- |
| Missing release scope | `gig` does not know which tickets belong to the release. | Use `--ticket-file`, saved snapshots, or a project-backed release scope. |
| Ambiguous topology | Branch order cannot be inferred safely. | Add `--from`, `--to`, or save project defaults. |
| Provider auth fails | The provider session is missing or expired. | Run `gig login`. |
| Verdict is `BLOCKED` | Expected evidence is missing from the target path. | Inspect the blocked ticket, then rerun verify. |
| Verdict is `WARNING` | Evidence exists, but risk needs review. | Review DB/config/dependency/Mendix notes before release. |

## Practical Rule

Do not start release day by writing config.
Start with project plus verify:

```bash
gig project use payments
gig verify --release rel-2026-04-09
```
