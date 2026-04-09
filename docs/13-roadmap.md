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

### Team Workflow Support

Shipped:

- config loading from `gig.yaml` style files
- environment and branch mapping from config
- repository catalog with service, owner, kind, and notes
- `manifest generate` for Markdown or JSON release packets
- `doctor` for config coverage and repo mapping checks

## What Comes Next

### Near-Term

- richer JSON contracts for downstream tooling
- stronger release packet structure and bundle-friendly output
- dependency trailer parsing
- ticket snapshots
- better multi-repo examples and docs

### After That

- Jira work-item enrichment
- PR and deployment evidence
- multi-ticket release bundles
- release-level planning such as `gig plan --release <release-id>`

### Later

- controlled promote execution
- safer backport or cherry-pick workflows
- rollback notes
- stronger reporting
- broader enterprise adapter coverage including SVN

## Release Philosophy

The project follows one rule:

safe release work comes before clever automation.
