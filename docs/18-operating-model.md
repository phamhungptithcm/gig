# Operating Model

## Goal

Define the operating concepts `gig` needs in order to work in real team release flows.

## Core Concepts

### Ticket Change Set

The complete set of evidence for one ticket across repositories.

Suggested contents:

- ticket ID
- repositories touched
- commits
- pull requests or merge requests
- builds
- deployments
- related tickets
- manual steps

### Environment State

The ticket's known state in each environment line.

Examples:

- discovered in branch only
- deployed to dev
- QA failed
- redeployed to test
- client review failed
- approved for production
- promoted to prod

### Release Manifest

A generated release packet for one ticket or a release bundle.

Current packet output already includes:

- source and target branch comparison
- risk summary
- manual steps
- QA checklist
- client review notes
- release manager checklist

Later it should also grow toward:

- multi-ticket bundles
- approval checklist history
- rollback notes
- external evidence from Jira, PRs, and deployments

### Risk Signal

Each promotion candidate should be classified, for example:

- `safe`
- `warning`
- `manual-review`
- `blocked`

Risk signals should come from facts such as:

- DB migrations present
- Mendix `.mpr` changes present
- config or secret changes detected
- dependency ticket missing
- repository unavailable
- target branch behind expected baseline

## Current Command Direction

The current command set already covers the core read-only workflow:

### `gig inspect <ticket-id>`

Show the full ticket change set across repositories.

### `gig env status <ticket-id>`

Show what is known about the ticket in each environment line.

### `gig plan --ticket <ticket-id> --from <branch> --to <branch>`

Build a promotion plan with risks, blockers, and manual steps.

### `gig verify`

Run pre-release checks and emit `safe / warning / blocked` outcomes.

### `gig manifest generate`

Produce a Markdown or JSON release packet for QA, client review, and release managers.

### `gig snapshot create --release <release-id>`

Capture a repeatable ticket baseline inside a named release snapshot set.

### `gig plan --release <release-id>`

Build a release-level plan by aggregating saved ticket snapshots.

### `gig doctor`

Check inferred topology, optional overrides, and repo catalog quality.

## Recommended Next Command Additions

- richer Jira, PR, and deployment evidence

## Current Config Direction

The current config already supports:

- repository catalog with service names and owners
- environment-to-branch mapping
- repository kind such as `app`, `db`, `mendix`, or `infra`
- simple repo notes that can appear in release packets

Next useful additions:

- manual-step definitions
- deployment source configuration
- issue tracker configuration
- pull request and deployment integration settings

## Recommended Commit And PR Metadata

Relying only on the subject line will be too weak in larger teams.

Support should grow toward commit trailers and PR metadata such as:

- `Depends-On`
- `Affects-Service`
- `Change-Type`
- `Manual-Step`
- `Risk`
- `DB-Migration`
- `Feature-Flag`

Git's trailer support makes this a practical path instead of a custom format.

## Integration Priority

The practical order should be:

1. Git history and patch-based comparison
2. repository and service catalog
3. release packet and stronger JSON output
4. Jira work item enrichment
5. GitHub or GitLab pull request and deployment evidence
6. multi-ticket release planning
7. controlled promote execution
8. SVN and advanced enterprise adapters

## Adoption Path For Teams

### Step 1

Use `gig` in read-only mode for ticket inspection and branch comparison.

### Step 2

Add a small config file so branch names, service names, and owners are clear.

### Step 3

Use `gig doctor` until the workspace and config are reliable.

### Step 4

Start generating release packets for QA, client review, and release managers.

### Step 5

Integrate Jira and deployment evidence.

### Step 6

Move toward release bundles and, later, controlled execution.
