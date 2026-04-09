# Product Strategy

## Recommended Positioning

`gig` should be positioned as:

`A ticket-aware release reconciliation and promotion planning tool for multi-repo teams.`

That positioning is stronger than "find commits by ticket."
It matches the real problem better and leaves room for integrations, release evidence, and safe automation.

## What `gig` Should Not Try To Replace

`gig` should not try to replace:

- Jira or Asana for task management
- GitHub, GitLab, or Bitbucket for PR reviews
- CI/CD systems for builds and deployments
- release managers for final approval on risky changes

Instead, it should connect those signals and make release decisions safer.

## Core Product Jobs

`gig` should help teams answer six practical questions:

1. Where did this ticket change code, scripts, or low-code assets?
2. What commits and pull requests belong to this ticket now?
3. What is already present in each branch or environment line?
4. What is still missing before this ticket can move safely?
5. What dependencies, risks, and manual steps must be included?
6. What evidence should be reviewed before approval and promotion?

## Primary Users

### Developer

Needs to confirm whether all follow-up commits for the ticket are accounted for.

### QA Or UAT Coordinator

Needs visibility into what changed since the last failed verification round.

### Release Engineer

Needs a promotion plan and a release packet, not just a list of commits.

### Delivery Lead Or Outsourcing Lead

Needs confidence that cross-repo tickets can be promoted without hidden omissions.

## Product Principles

### 1. Read Evidence, Do Not Guess

Whenever possible, `gig` should use evidence from:

- commit history
- pull requests or merge requests
- build status
- deployment records
- issue tracker links

### 2. Separate Branch State From Environment State

A commit being present in a branch is not the same as a ticket being deployed and verified in an environment.

### 3. Safe By Default

The product should prefer:

- dry-run
- warnings
- risk classification
- manual approval

over clever execution.

### 4. Human Output And Machine Output Must Both Matter

Terminal output is useful for developers.
JSON and release manifests are required for automation, CI/CD, and audit trails.

### 5. Adopt The Team's Current Workflow First

Many teams already use:

- `dev -> test -> prod`
- `staging -> main`
- backport or cherry-pick promotion

`gig` should improve those workflows before trying to force a process rewrite.

## Success Metrics

The product should help teams improve:

- missed-commit incidents during promotion
- time spent preparing a release
- time spent reconciling ticket changes across repositories
- release failures caused by missing dependent changes
- auditability of what went to each environment

The broader delivery metrics can still align with DORA-style outcomes such as deployment frequency, lead time, and change failure rate, but `gig` should focus on the release-reconciliation slice of that problem.

## Product Maturity Model

### Stage 1: Visibility

- scan repositories
- find ticket commits
- compare source and target branches

### Stage 2: Planning

- inspect a ticket across repositories
- detect missing items and likely dependencies
- classify risks and manual steps

### Stage 3: Evidence

- emit JSON and release manifests
- attach pull request, build, and deployment evidence
- show environment-aware status

### Stage 4: Controlled Execution

- produce a promotion plan
- require confirmation and approvals
- execute only safe and supported steps

### Stage 5: Enterprise Coverage

- SVN support
- Jira enrichment
- Mendix and other high-risk change heuristics
- stronger reporting for mixed environments
