# Test Strategy

## Goal

Keep the release workflow logic safe to change.

The tests should tell us quickly if we broke:

- repo discovery
- ticket parsing
- branch comparison
- environment status
- release verification
- promotion planning output
- config loading
- release packet rendering
- doctor checks

## Main Test Types

### Unit Tests

Use unit tests for:

- ticket parsing
- repo detection
- diff orchestration
- inspect and plan logic
- config parsing

### Adapter Tests

Use local temporary Git repos to test real Git behavior.

This matters because release mistakes usually come from real branch history, not from fake string matching.

### Golden CLI Tests

Use golden files to keep human-readable output stable.

Current good targets:

- `scan`
- `find`
- `diff`
- `inspect`
- `env status`
- `verify`
- `plan`
- `manifest`
- `doctor`

### JSON Contract Tests

JSON output should stay stable enough for scripts and CI tooling.

Current good targets:

- `plan --format json`
- `doctor --format json`
- `manifest --format json`

## Current Coverage

- `internal/cli`
  - golden output tests for the main commands
  - help and usage behavior
- `internal/repo`
  - recursive discovery
  - enclosing repo detection
- `internal/ticket`
  - ticket extraction, normalization, and validation
- `internal/config`
  - config loading and repo lookup
- `internal/diff`
  - compare orchestration
- `internal/inspect`
  - risk-signal inference
  - environment-state logic
- `internal/plan`
  - verdict logic
  - promotion plan generation
  - verification generation
- `internal/scm/git`
  - real Git search and compare behavior

## What We Still Want Next

- stronger JSON contract coverage
- more release packet scenarios
- more mixed-branch and multi-repo scenarios
- integration-focused tests once Jira and deployment evidence are added
