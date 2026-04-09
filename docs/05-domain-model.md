# Domain Model

This page describes the main data shapes `gig` works with today.

The goal is simple:
the CLI, services, and output code should all talk about the same things in the same way.

## Repository

Represents one detected repository in the workspace.

Fields:

- `Name`
- `Root`
- `Type`
- `CurrentBranch`

## Commit Ref

Represents one commit that `gig` found for a ticket.

Fields:

- `Hash`
- `Subject`
- `Branches`

## Ticket Search Result

Used by `gig find`.

Contains:

- repository
- list of matching commits

## Branch Diff

Used by `gig diff`.

Contains:

- ticket ID
- source branch
- target branch
- commits found in the source branch
- commits found in the target branch
- commits missing in the target branch

## Risk Signal

Used by `gig inspect`, `gig verify`, and `gig plan`.

Represents something that may need extra review.

Examples:

- `db-change`
- `config-change`
- `mendix-model`

Fields:

- `Code`
- `Level`
- `Summary`
- `Examples`

## Repository Inspection

Used by `gig inspect`.

Represents the full ticket picture for one repository.

Contains:

- repository
- commits
- branches seen
- risk signals

## Environment

Represents one logical environment in the release flow.

Examples:

- `dev`
- `test`
- `uat`
- `prod`

Each environment maps to one branch name.

## Environment Result

Used by `gig env status`, `gig verify`, and `gig plan`.

Represents the ticket state in one environment branch.

Current states:

- `present`
- `aligned`
- `behind`
- `not-present`
- `branch-missing`

## Promotion Plan

Used by `gig plan`.

This is a read-only plan.
It does not execute any Git write action.

Contains:

- ticket ID
- source branch
- target branch
- environment mapping
- overall summary
- overall verdict
- per-repo plans

Each repo plan can include:

- compare result
- risk signals
- environment statuses
- manual steps
- planned actions
- notes
- verdict

## Verification Result

Used by `gig verify`.

Contains:

- overall verdict
- summary counts
- human-readable reasons
- per-repo checks
- manual steps

## Why These Models Matter

- the terminal output stays consistent
- JSON output stays close to human output
- new commands can build on the same service results
- future release packet and config work has a stable base
