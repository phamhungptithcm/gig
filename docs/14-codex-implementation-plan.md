# Codex Implementation Plan

## Purpose

This file is for Codex-driven implementation work.

It defines:

- task breakdown
- implementation order
- definition of done
- file ownership by area

## Implementation Order

### Step 1: Foundation

Tasks:

- create repo structure
- wire CLI bootstrap
- define SCM interface
- implement repository scanner

Main files:

- `cmd/gig/main.go`
- `internal/cli/*`
- `internal/repo/*`
- `internal/scm/*`

### Step 2: MVP Commands

Tasks:

- implement `scan`
- implement `find`
- implement `diff`
- add output renderer

Main files:

- `internal/ticket/*`
- `internal/diff/*`
- `internal/output/*`
- `internal/cli/*`

### Step 3: Tests

Tasks:

- unit tests
- Git adapter tests
- CLI output tests later

Main files:

- `internal/**/*_test.go`

### Step 4: Promote Design

Tasks:

- define promotion models
- define dry-run flow
- define confirmation flow

Main files:

- `internal/promote/*`
- `docs/07-promotion-engine.md`

### Step 5: Advanced Features

Tasks:

- dependency parsing
- snapshot support
- SVN adapter
- Jira and Mendix extensions

Main files:

- `internal/dependency/*`
- `internal/snapshot/*`
- `internal/scm/svn/*`

## Definition Of Done

A task is done when:

- code builds
- tests pass
- behavior is documented
- output is readable
- no hidden destructive action is introduced
- package boundaries stay clean

## File Ownership By Area

### CLI

- `cmd/gig/*`
- `internal/cli/*`

### Domain And SCM Contracts

- `internal/scm/types.go`
- `internal/scm/registry.go`

### Git Adapter

- `internal/scm/git/*`

### SVN Adapter

- `internal/scm/svn/*`

### Repository Discovery

- `internal/repo/*`

### Ticket Search

- `internal/ticket/*`

### Branch Diff

- `internal/diff/*`

### Output

- `internal/output/*`

### Future Promotion Work

- `internal/promote/*`
- `internal/dependency/*`
- `internal/snapshot/*`

## Current Status

Implemented now:

- foundation structure
- Git-first scan
- ticket find
- branch diff
- core tests

Planned next:

- config loading
- JSON output
- promote dry-run
- richer dependency logic
