# Roadmap

## Direction In One Sentence

`gig` is moving from a ticket commit helper into a ticket-aware release workflow tool for multi-repo teams.

## What Is Already Here

### Base CLI

Shipped:

- `scan`
- `find`
- `diff`
- `version`

### Release Visibility

Shipped:

- `inspect`
- `env status`
- risk signals for DB, config, and Mendix-style changes

### Read-Only Release Decisions

Shipped:

- `verify`
- `plan`
- JSON output for release planning and review

## What Comes Next

### Near-Term

- config loading
- environment and branch mapping from config
- repository or service catalog
- richer release manifest output
- `manifest generate`
- `doctor`

### After That

- dependency trailer parsing
- ticket snapshots
- Jira work-item enrichment
- PR and deployment evidence
- multi-ticket release bundles

### Later

- controlled promote execution
- safer backport or cherry-pick workflows
- rollback notes
- stronger reporting
- SVN support

## Release Philosophy

The project follows one rule:

safe release work comes before clever automation.
