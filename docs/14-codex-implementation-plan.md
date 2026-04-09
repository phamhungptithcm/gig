# Engineering Implementation Plan

## Purpose

This page tracks the internal implementation direction for the codebase.

It is mostly useful for maintainers.

## What Is Implemented Now

- repository scanning
- Git-first SCM abstraction
- ticket search
- branch diff
- ticket inspection across repos
- environment status
- release verification
- read-only promotion planning
- JSON output for plan and verify
- CLI golden tests

## What The Next Engineering Slice Should Cover

- config loading
- environment and branch mapping from config
- repository catalog
- richer manifest generation
- better JSON contracts
- a `doctor` command

## Definition Of Done

A change is done when:

- the code builds
- tests pass
- docs match the real behavior
- output is easy to read
- no hidden destructive action is introduced

## Main Code Areas

- CLI: `cmd/gig/*`, `internal/cli/*`
- Repo discovery: `internal/repo/*`
- Ticket search: `internal/ticket/*`
- Branch diff: `internal/diff/*`
- Ticket inspection: `internal/inspect/*`
- Release planning and verification: `internal/plan/*`
- Output: `internal/output/*`
- Git adapter: `internal/scm/git/*`
- Shared SCM contracts: `internal/scm/*`
