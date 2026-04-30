# Branching Model

`gig` needs a promotion path to answer release-readiness questions.

For example:

```text
staging -> main
```

## Remote Mode

Remote mode tries provider-backed branch metadata first:

- protected branches
- default branch
- provider branching model when available
- high-confidence environment order

If confidence is high:

```bash
gig verify ABC-123 --repo github:owner/name
```

If confidence is low, `gig` asks for explicit branches:

```bash
gig verify ABC-123 --repo github:owner/name --from staging --to main
```

## Local Mode

Local mode does not always have enough provider metadata.

Use:

```bash
gig verify ABC-123 --path . --from staging --to main
```

or save defaults:

```bash
gig project add local --path . --from staging --to main --use
```

## Environment Mapping

Use `--envs` when your environment names do not map cleanly:

```bash
gig verify ABC-123 --repo github:owner/name --envs dev=develop,test=staging,prod=main --from staging --to main
```

## Good Branch Inputs

Good:

- `develop -> staging`
- `staging -> main`
- `release/2026.04 -> trunk`
- saved project branch defaults

Risky:

- relying on local mode without `--from` and `--to`
- assuming `release/*` means the same thing for every team
- expecting `gig` to guess when protected branches conflict

## Rule

If `gig` says it is not sure, treat that as correct behavior.
Give it the branch path or save a project.
