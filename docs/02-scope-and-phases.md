# Scope And Phases

## Phase 0: Foundation

Goal:
Create a clean base that can grow without a rewrite.

Scope:

- project skeleton
- CLI bootstrap
- package boundaries
- repository discovery
- Git detection
- path scanning
- basic tests for core helpers

Output:

- a runnable CLI
- clean internal modules
- Git-first SCM abstraction

## Phase 1: MVP CLI

Goal:
Help teams inspect ticket-related changes before promotion.

Scope:

- `gig scan`
- `gig find <ticket-id>`
- `gig diff --ticket <ticket-id> --from <branch> --to <branch>`
- human-readable output
- test coverage for parser, discovery, diff logic, and Git adapter behavior

Output:

- workspace scan
- commit search by ticket
- branch gap detection by ticket

## Phase 2: Promote Automation

Goal:
Move from read-only analysis to guided promotion.

Scope:

- `gig promote`
- promotion plan
- dry-run mode
- confirmation step before write actions
- early conflict checks

Output:

- safe promotion plan
- clear preview of what will happen
- optional execution after confirmation

## Phase 3: Dependency And Snapshot

Goal:
Make the tool smarter about ticket relationships and repeatable release state.

Scope:

- parse dependency footers such as `depends-on: XYZ-456`
- include related ticket changes in planning
- snapshot commits by ticket and branch state

Output:

- better promotion safety
- better audit support

## Phase 4: SVN, Jira, And Mendix Advanced Features

Goal:
Support real enterprise environments more fully.

Scope:

- working SVN adapter
- Jira integration
- Mendix warnings and higher-risk file detection
- stronger release reporting

Output:

- broader SCM support
- better enterprise workflow coverage

## What Is Not In MVP

- automatic production promotion by default
- silent destructive actions
- auto-resolution of complex merge conflicts
