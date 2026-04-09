# Roadmap

## Strategic Direction

`gig` should evolve from a read-only ticket commit helper into a ticket-aware release reconciliation and promotion planning tool.

The roadmap should therefore optimize for five pillars:

- visibility across repositories
- environment-aware release evidence
- safer promotion planning
- machine-readable release outputs
- enterprise and mixed-tooling support

## v0.1.x: Stabilize The MVP

- harden `scan`, `find`, and `diff`
- improve CLI output and regression coverage
- keep Git-first repository discovery stable
- tighten docs, releases, and project automation

## v0.2.x: Add Data Contracts And Team Configuration

- config loading
- repository or service catalog
- environment-to-branch mapping
- JSON output for CI and automation
- better error models and report structure

## v0.3.x: Add Ticket Inspection And Release Planning

- `inspect`-style ticket view across repositories
- richer branch comparison results
- release manifest generation
- dry-run promotion plan preview
- clearer risk, warning, and blocker output

## v0.4.x: Add Evidence And Dependency Awareness

- dependency trailer parsing
- ticket snapshot support
- pull request or merge request evidence
- deployment or environment evidence
- multi-ticket release bundle planning

## v0.5.x: Add Controlled Promotion Execution

- confirmation-gated promote execution
- safer cherry-pick or backport workflows
- execution reporting
- rollback notes and manual follow-up guidance

## v0.6.x: Expand Enterprise Coverage

- SVN implementation
- Jira enrichment
- Mendix-specific risk heuristics
- stronger release reporting for mixed environments

## General Direction

Each release should keep one rule:
safe release work comes before clever automation.
