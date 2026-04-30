# Release Packet

`gig packet` turns audit evidence into a release packet.

Use it when the output needs to be shared with QA, release managers, clients, or automation.

`gig manifest ...` still works as a compatibility alias.

## One Ticket

```bash
gig packet ABC-123
```

If you are outside the checkout:

```bash
gig packet ABC-123 --repo github:owner/name
```

If branch topology is not inferable:

```bash
gig packet ABC-123 --from staging --to main
```

## Ticket File

```bash
gig packet --ticket-file tickets.txt
```

Use this when a release contains a known ticket list.

## Saved Release

```bash
gig packet --release rel-2026-04-09 --path .
```

Use this when release scope is already saved through snapshots or project-backed release flow.

## Output Formats

Markdown for humans:

```bash
gig packet ABC-123
```

JSON for tooling:

```bash
gig packet ABC-123 --json
```

## What A Good Packet Contains

- ticket scope
- repo evidence
- promotion path
- verdict and notes
- missing commits or dependency risk
- manual-review hints
- provider evidence when available

## Common Problems

| Problem | Next action |
| --- | --- |
| Packet lacks branch context | Add `--from` and `--to`, or use a project. |
| Packet lacks provider evidence | Confirm provider login and remote target. |
| Packet is too broad | Use a single ticket or smaller ticket file. |
| Automation needs stable output | Use `--json` and track the JSON reference. |
