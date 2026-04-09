# Scope And Phases

## What Exists Now

The current project already covers the core read-only release workflow.

### Foundation

Shipped:

- CLI bootstrap
- package boundaries
- repository discovery
- Git-first SCM abstraction
- test coverage for the core layers

### Visibility And Verification

Shipped:

- `gig scan`
- `gig find`
- `gig diff`
- `gig inspect`
- `gig env status`
- `gig verify`
- `gig plan`
- JSON output for `verify` and `plan`
- basic risk signals for DB, config, and Mendix-style changes

This is the current MVP that teams can actually try for real release checks.

## What The Current MVP Is Meant To Do

The job of the current MVP is simple:

- help teams see the full ticket story
- show where a ticket is behind in the environment flow
- give a clear go or no-go signal before promotion
- produce a release plan without changing repos

## What Comes Next

### Next Release Focus

- config loading
- environment and branch mapping from config
- repository or service catalog
- richer release manifest output
- a dedicated `manifest generate` command
- a `doctor` command for branch and ticket hygiene

### Later Release Focus

- dependency trailer parsing
- ticket snapshots
- Jira and PR evidence
- deployment evidence
- multi-ticket release bundles
- controlled promote execution after strong dry-run and approval flows

### Enterprise Coverage Later

- SVN support
- stronger Mendix support
- mixed-tooling workflow coverage

## What Is Deliberately Out Of Scope Today

- silent write actions
- automatic production promotion
- auto-resolution of complex merge conflicts
- pretending branch presence is the same as deployment evidence
