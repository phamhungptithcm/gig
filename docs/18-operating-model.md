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

Suggested contents:

- ticket list
- repository list
- commits or PRs included
- source and target branch comparison
- dependency links
- risk summary
- manual steps
- approval checklist
- rollback notes

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

## Recommended Command Direction

The current commands are a good MVP base, but the product should grow toward:

### `gig inspect <ticket-id>`

Show the full ticket change set across repositories.

### `gig env status <ticket-id>`

Show what is known about the ticket in each environment line.

### `gig plan --ticket <ticket-id> --from <branch> --to <branch>`

Build a promotion plan with risks, blockers, and manual steps.

### `gig plan --release <release-id>`

Build a release plan for multiple tickets together.

### `gig manifest generate`

Produce a JSON or Markdown release packet for review and audit.

### `gig verify`

Run pre-release checks and emit `safe / warning / blocked` outcomes.

### `gig doctor`

Check branch naming, ticket discipline, config quality, and missing integration data.

## Recommended Config Direction

The current config spec should expand beyond simple scanner defaults.

Important additions:

- repository catalog with service names and owners
- environment-to-branch mapping
- repository type or risk type such as `app`, `db`, `mendix`, `infra`
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
3. JSON output and release manifest generation
4. Jira work item enrichment
5. GitHub or GitLab pull request and deployment evidence
6. controlled promote execution
7. SVN and advanced enterprise adapters

## Adoption Path For Teams

### Step 1

Use `gig` in read-only mode for ticket inspection and branch comparison.

### Step 2

Standardize branch names, commit messages, and trailers.

### Step 3

Add repo catalog and environment mapping config.

### Step 4

Start generating release manifests and review packets in CI.

### Step 5

Integrate Jira and deployment evidence.

### Step 6

Enable controlled promotion planning and, later, controlled execution.
