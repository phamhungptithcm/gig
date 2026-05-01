# Command Reference

This is the short command reference. For the learning path, start with the [Quick Start](19-quickstart.md).

## Core

```bash
gig
gig login
gig ABC-123
gig inspect ABC-123
gig verify ABC-123
gig packet ABC-123
```

Use `--repo github:owner/name` when you are outside the checkout or overriding the inferred remote target.

## Projects

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
gig project list
gig project use payments
gig project show
```

`gig workarea ...` remains available as a compatibility alias.

## Release

```bash
gig verify --ticket-file tickets.txt
gig packet --ticket-file tickets.txt
gig verify --release rel-2026-04-09 --path .
gig packet --release rel-2026-04-09 --path .
```

## Local Fallback

```bash
gig repos --path .
gig ABC-123 --path .
gig verify ABC-123 --path . --from staging --to main
gig packet ABC-123 --path . --from staging --to main
```

## Advanced Inspection

```bash
gig commits ABC-123 --path .
gig where ABC-123 --project payments
gig diff --ticket ABC-123 --from dev --to test --path .
gig plan ABC-123
gig snapshot create ABC-123 --path . --from staging --to main
gig doctor
```

Compatibility aliases: `scan`, `find`, `env status`, and `manifest`.

## AI Assist

```bash
gig assist doctor
gig assist setup
gig explain ABC-123 --audience release-manager
gig assist release --release rel-2026-04-09 --audience release-manager
gig ask "what is still blocked?"
gig resume
```

## Conflict Tools

```bash
gig resolve status --path .
gig resolve start --path .
gig assist resolve --path . --ticket ABC-123
```

## Install And Update

```bash
gig version
gig update
gig update v2026.5.0
gig update v2026.5.1
```

## Common Flags

| Flag | Use |
| --- | --- |
| `--repo` | Live remote repository target when it cannot be inferred. |
| `--path` | Local workspace fallback. |
| `--project`, `--workarea` | Saved project context. |
| `--from`, `--to` | Explicit promotion path. |
| `--envs` | Environment-to-branch mapping. |
| `--ticket-file` | Batch ticket input. |
| `--release` | Saved release scope. |
| `--json`, `--format json` | Automation output. |
| `--config` | Optional override file. |
