# Test Strategy

## Goals

- keep parsing and orchestration testable without invoking the CLI
- verify repository discovery across nested workspace layouts
- exercise Git adapter behavior with real temporary repositories
- keep output stable and easy to review

## Test Types

### Unit Tests

Use unit tests for:

- ticket parsing
- repo detection
- diff orchestration
- config parsing later

### Adapter Contract Tests

Use shared contract-style tests to make sure each SCM adapter behaves the same way for the same use cases.

This matters when SVN is added later.

### Golden File Tests For CLI Output

Use golden files to verify human-readable CLI output stays stable.

Good targets:

- `scan` output
- `find` output
- `diff` output
- future `promote --dry-run` output

### Integration Tests With Local Git Repos

Use temporary local Git repositories created during tests.

Test flows such as:

- finding commits by ticket
- comparing branches
- detecting missing commits
- confirming cherry-picked commits are no longer missing

## Test Style

- use temp directories for filesystem-driven tests
- use local Git config inside temp repos for adapter tests
- keep service tests independent from terminal rendering

## Current Coverage

- `internal/repo`
  - recursive discovery
  - enclosing repo detection
- `internal/ticket`
  - regex-based ticket extraction, normalization, and validation
- `internal/diff`
  - service orchestration and result filtering with fake adapters
- `internal/scm/git`
  - search by ticket in a real temp repo
  - compare branches and confirm missing commit detection
  - confirm cherry-picked commits are no longer reported as missing

## Planned Additions

- renderer snapshot tests
- golden file tests for CLI output
- config loading tests once config files exist
- JSON output contract tests
- promotion dry-run integration tests
