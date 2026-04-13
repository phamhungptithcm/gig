---
name: gig-product-guardrails
description: Use this skill when changing product-facing behavior in the gig repository. It keeps agents aligned with gig's remote-first, zero-config-first, deterministic release-audit direction. Trigger when working on CLI design, provider-backed flows, docs changes, or AI integration boundaries.
---

# gig Product Guardrails

This skill keeps implementation work aligned with the product direction of `gig`.

## Product Direction

`gig` is moving toward:

- source-control-native access
- remote-first usage
- zero-config first success
- deterministic release evidence with optional AI briefings

## Required Biases

- Prefer provider-backed flows over local-only assumptions.
- Infer topology from protected branches when possible.
- Keep config as an optional enhancement, not a first-run burden.
- Keep CLI commands intent-driven and readable from the terminal.
- Keep AI additive. It should explain `gig` evidence, not replace it.

## Implementation Rules

- Keep `cmd/gig` and `internal/cli` thin.
- Put release behavior in `internal/*` services.
- Put provider logic in `internal/scm/*`.
- Do not leak provider-specific language into generic output unless it helps the user materially.

## Docs And Verification

When user-facing behavior changes, update:

- `README.md`
- `docs/03-cli-spec.md`
- `docs/19-quickstart.md`
- `docs/00-product-overview.md`
- `docs/13-roadmap.md`
- `docs/22-product-reset-audit.md`

Then run:

```bash
go test ./internal/cli ./internal/assistant
./.venv-docs/bin/mkdocs build --strict
```
