# Promotion Engine

## Current Status

The promotion engine is partially implemented in read-only form.

Today, `gig` can already:

- inspect ticket scope across repos
- compare source and target branches
- detect when the source branch is behind an earlier environment
- classify a promotion as `safe`, `warning`, or `blocked`
- generate a read-only promotion plan in human or JSON output

Today, it does not execute any Git write action.

## Goal

Move from:

`show me what is missing`

to:

`help me move the right changes safely`

## What The Current Read-Only Engine Does

### 1. Collect Ticket Evidence

Find all commits for the requested ticket across detected repositories.

### 2. Read Environment State

Check how the ticket appears in the configured environment flow such as:

- `dev`
- `test`
- `prod`

### 3. Compare Source And Target

Compare the selected source branch and target branch for each repository.

### 4. Infer Risk Signals

Look for file patterns that probably need manual review, such as:

- DB changes
- config changes
- Mendix model changes

### 5. Build A Verdict

Return one of:

- `safe`
- `warning`
- `blocked`

### 6. Build A Plan

Produce a read-only plan that shows:

- missing commits
- manual steps
- repo-level notes
- planned actions

## Current Blockers

Right now, the engine blocks a repo when:

- the source branch does not exist
- the target branch does not exist
- the selected source branch is behind an earlier environment
- the selected source branch does not actually contain ticket commits

## What Comes Next

Later versions should add:

- dependency-aware checks
- release packet generation in richer formats
- deployment evidence
- Jira or PR evidence
- controlled execution after dry-run and approval

## Design Principles

- safe by default
- read facts first
- no hidden write actions
- make human review easier
- keep machine-readable output aligned with terminal output
