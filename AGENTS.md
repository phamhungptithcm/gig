# AGENTS.md

## Purpose

This file tells coding agents how to work effectively in the `gig` repository.

`gig` is a Go CLI for ticket-aware release inspection, promotion planning, verification, and release packet generation across multiple repositories and source-control systems.

The product is in an active transition from:

- local-workspace-first

to:

- source-control-native
- remote-first
- zero-config-first

Agents working here should preserve that direction.

## Product Reality

Current stable strengths:

- ticket-aware commit discovery
- branch-to-branch reconciliation
- risk and manual-step inference
- repeatable snapshots
- release-level planning and bundle generation
- human-readable and JSON output

Current transition state:

- local Git and local SVN flows are mature
- live direct GitHub support is being introduced
- GitLab, Bitbucket, and remote SVN provider flows are not complete yet
- config still exists, but it should become optional enhancement, not first-run burden

Do not write code that pushes the product back toward config-heavy or local-only assumptions.

## Engineering Priorities

When making product-facing changes, prefer this order:

1. source-control-native access
2. zero-config first success
3. protected-branch or topology auto-detection
4. clear CLI intent
5. backward compatibility for existing local workflows
6. richer evidence and automation

Good changes:

- add provider-backed flows that work without cloning
- infer defaults from source-control metadata
- reduce required flags
- keep CLI output understandable to humans
- keep JSON useful for tooling

Bad changes:

- requiring `gig.yaml` for first-run success
- adding more mandatory `--from`, `--to`, or `--envs` flows where source control can answer directly
- leaking provider-specific logic into output or product terminology
- rewriting local Git or SVN flows unnecessarily when adding remote support

## Architecture Rules

Keep the CLI thin.

Use the existing layering:

- `cmd/gig`
  CLI entrypoint
- `internal/cli`
  argument parsing, usage, runtime wiring, command orchestration
- `internal/*` services
  real release use cases
- `internal/scm/*`
  SCM and provider-specific adapters
- `internal/output`
  human and JSON rendering

When adding remote provider support:

- prefer a clean provider or session layer over stuffing more logic into local Git or SVN adapters
- keep ticket, inspect, plan, and manifest services source-agnostic
- treat repository targets as inputs to the engine, not as special cases in output rendering
- do not break local workspace scanning while remote flows are still incomplete

## Source-Control-Native Guidance

The target product shape is:

- `gig login <provider>`
- `gig inspect <ticket> --repo <provider-target>`
- `gig verify --ticket <ticket> --repo <provider-target>`
- `gig plan --ticket <ticket> --repo <provider-target>`
- `gig manifest generate --ticket <ticket> --repo <provider-target>`

When extending this direction:

- prefer vertical slices that are truly usable end to end
- implement one provider completely enough to ship, then expand
- infer branch topology from protected branches when possible
- fall back to explicit flags only when topology is ambiguous
- keep local mode as a supported fallback, not the main mental model

If you add GitLab, Bitbucket, or remote SVN:

- mirror the GitHub live slice structure where practical
- add auth or session handling first
- make the login or access failure actionable
- do not claim support in docs until the command path is actually wired and tested

## CLI Rules

This repo is CLI-first.
CLI changes must feel practical from the terminal.

When changing commands:

- keep help text short and direct
- prefer intent-driven wording over engine-internal wording
- avoid adding new mandatory flags unless unavoidable
- update usage text in `internal/cli/app.go`
- update README and `docs/03-cli-spec.md` when user-facing behavior changes

If a command supports both local and remote modes:

- keep both modes explicit and testable
- make error messages tell the user what to do next
- auto-login or auto-detect only when behavior is predictable

## Output Rules

Human output and JSON output both matter.

When changing output:

- preserve stable phrasing where possible
- update golden tests intentionally
- keep JSON contracts additive when possible
- avoid provider-specific noise unless it materially helps the user

If a remote repository target is used:

- show a clear scope label such as `github:owner/name`
- do not pretend it is a local workspace path

## Testing Rules

This project relies on:

- unit tests
- real adapter tests for local Git and SVN behavior
- golden tests for CLI output
- JSON contract checks where relevant

Minimum expectations for agent changes:

- add or update unit tests for new logic
- add or update CLI tests for new command behavior
- update golden files only when the output change is intentional
- run focused tests first, then `go test ./...` before finishing if the change is meaningful

For provider-backed remote flows:

- prefer deterministic fake command runners or fake CLIs in tests
- do not require real network calls in normal test runs
- cover auth-failure and auto-login behavior when introduced

## Docs Rules

This repo is docs-heavy by design.

If your change alters product behavior, also update the relevant docs:

- `README.md`
- `docs/03-cli-spec.md`
- `docs/19-quickstart.md`
- `docs/00-product-overview.md`
- `docs/13-roadmap.md`
- `docs/22-product-reset-audit.md` when the change materially advances or changes the reset direction

Run:

```bash
./.venv-docs/bin/mkdocs build --strict
```

when docs or user-facing behavior change.

## Safe Change Strategy

Prefer these patterns:

1. add the new provider or capability behind a clean interface
2. wire one command path end to end
3. test the full slice
4. update docs
5. expand to the next command or provider

Avoid large rewrites that:

- change local Git and SVN behavior at the same time as remote provider introduction
- mix command redesign, provider auth, output redesign, and planning logic in one change
- force a new architecture across the entire repo before a usable slice exists

## Practical Commands

Common commands for agents working here:

```bash
gofmt -w <files>
go test ./...
./.venv-docs/bin/mkdocs build --strict
```

Useful focused test runs:

```bash
go test ./internal/cli
go test ./internal/scm/...
go test ./internal/inspect ./internal/plan ./internal/snapshot
```

## Git And PR Expectations

Use branch names under `hunpeolabs/` unless told otherwise.

Use Conventional Commits, for example:

- `feat(cli): add remote github inspect flow`
- `fix(plan): infer protected branch source for remote repos`
- `docs(strategy): clarify source-control-native direction`

PRs should be small enough to explain clearly.
For larger direction changes, land a vertical slice plus docs, not a vague framework.

## Final Reminder

The main failure mode in this repo is building something architecturally interesting but operationally awkward.

Bias toward:

- fewer flags
- clearer commands
- direct source-control access
- stable release reasoning
- incremental vertical slices

That is what fits `gig`.
