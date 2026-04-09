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

## Main Test Types

### Unit Tests

Use unit tests for:

- ticket parsing
- repo detection
- diff orchestration
- inspect and plan logic
- config parsing later

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

### JSON Contract Tests

JSON output should stay stable enough for scripts and CI tooling.

Current good target:

- `plan --format json`

## Current Coverage

- `internal/cli`
  - golden output tests for the main commands
  - help and usage behavior
- `internal/repo`
  - recursive discovery
  - enclosing repo detection
- `internal/ticket`
  - ticket extraction, normalization, and validation
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
- config loading tests once config exists
- release manifest tests
- more mixed-branch and multi-repo scenarios
