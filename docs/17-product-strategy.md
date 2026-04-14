# Product Strategy

## Recommended Positioning

`gig` should be positioned as:

`A remote-first release audit CLI for ticket-based delivery across many repositories.`

That is sharper than "find commits by ticket" and more honest than "generic release tool."
It centers the real job:
inspect ticket evidence, spot what is missing, and decide whether the next move is safe.

## The Core Promise

After install, `gig` should be usable immediately.
The user should not need to prepare a local workspace, write config, or memorize branch mappings before first value.

The shortest successful path should feel like this:

1. run `gig`
2. log in to a provider if needed
3. inspect a ticket
4. verify the next move
5. export a release packet when needed

## What Problem It Actually Solves

The real pain is not "I cannot grep commits."
The real pain is:

- one ticket touches many repos
- the ticket fails QA or client review and gets follow-up fixes
- branch topology differs across projects
- one missing commit or DB change can break a release
- users work across many clients or products and cannot reconfigure every time

`gig` should collapse that complexity into one readable audit workflow.

## Primary Users

### Developer Or Tech Lead

Needs to confirm whether all follow-up commits for the ticket are accounted for.

### QA Or UAT Coordinator

Needs visibility into what changed since the last failed verification round.

### Release Engineer

Needs a release verdict and a release packet, not just a list of commits.

### Delivery Lead Or Outsourcing Lead

Needs confidence that cross-repo tickets can be promoted without hidden omissions.

## Product Principles

### 1. Install Must Lead To First Value Fast

The product should assume that first-run friction kills adoption.
Install, login, inspect, and verify must be the main path.

### 2. Remote Evidence First, Local Evidence Second

Whenever possible, `gig` should prefer:

- remote provider APIs
- protected-branch discovery
- online branch and PR evidence

Local Git or SVN should stay available, but as a fallback or advanced mode.

### 3. Config Is An Override, Not A Gate

Config should improve quality, not unlock basic usefulness.
The default path should work without team-maintained setup.

### 4. Workareas Are A Core Product Primitive

Users often work across many products, clients, or release streams.
`gig` should let them save a project as a workarea with:

- provider connection
- repo scope
- inferred branch topology
- optional naming and notes
- preferred output depth

The next time they open `gig`, they should choose the workarea and continue.

### 5. Console UX Must Be Good Enough To Live In

The terminal experience should be:

- keyboard-first
- readable under multi-repo complexity
- summary first, detail on demand
- visually consistent across inspect, verify, and release views
- explicit about risk, missing evidence, and manual review items

### 6. Read Evidence, Do Not Guess

Whenever possible, `gig` should use evidence from:

- commit history
- pull requests or merge requests
- build status
- deployment records
- issue tracker links when available

### 7. Separate Branch State From Environment State

A commit being present in a branch is not the same as a ticket being deployed and verified in an environment.

### 8. Safe By Default

The product should prefer:

- dry-run
- warnings
- risk classification
- manual approval

over clever execution.

### 9. Human Output And Machine Output Must Both Matter

Terminal output is useful for developers.
JSON and release manifests are required for automation, CI/CD, and audit trails.

### 10. Improve Existing Delivery Workflows First

Many teams already use:

- `dev -> test -> prod`
- `staging -> main`
- backport or cherry-pick promotion

`gig` should improve those workflows before trying to force a process rewrite.

## The Product Jobs

`gig` should help teams answer these practical questions:

1. Which repos and remote branches does this ticket actually touch?
2. What commits, PRs, and follow-up fixes belong to it now?
3. What is already present in the target release path?
4. What is still missing before it can move safely?
5. What risks or manual checks still need human review?
6. What evidence should be saved for audit, QA, UAT, and release approval?

## What `gig` Should Not Try To Replace

`gig` should not try to replace:

- Jira or Asana for task management
- GitHub, GitLab, or Bitbucket for code review
- CI/CD systems for builds and deployments
- release managers for final approval on risky changes

It should connect those signals and reduce manual reconciliation work.

## UX Direction

The console UX should revolve around a few durable user intents:

- open or switch workarea
- inspect ticket
- verify release readiness
- capture release evidence
- export human or machine output

The UI should avoid making new users choose among many engine-shaped commands before they understand the result they need.

## Success Metrics

The product should help teams improve:

- missed-commit incidents during promotion
- time to first useful audit after install
- time spent preparing a release
- time spent reconciling ticket changes across repositories
- release failures caused by missing dependent changes
- repeated setup work across projects
- auditability of what went to each environment

The broader delivery metrics can still align with DORA-style outcomes such as deployment frequency, lead time, and change failure rate, but `gig` should focus on the release-reconciliation slice of that problem.

## Product Maturity Model

### Stage 1: Activation

- install and run immediately
- guided login
- remote repo targeting
- zero-config first success

### Stage 2: Audit

- inspect a ticket across repositories and branches
- detect missing items and likely dependencies
- classify risks and manual steps

### Stage 3: Workarea UX

- remember projects as workareas
- provide a strong console workflow for repeated use
- reduce repeated flag entry and setup friction

### Stage 4: Evidence

- emit JSON and release manifests
- attach pull request, build, and deployment evidence
- show environment-aware status

### Stage 5: Controlled Execution And Coverage

- produce promotion plans from stronger evidence
- require confirmation and approvals
- add broader provider coverage and enterprise edge cases

## Where The Project Is Now

Today, `gig` already has strong pieces of Stages 1 through 3, with the biggest remaining gap being polish and depth rather than direction.

Already available:

- ticket inspection across repos
- remote inspection across GitHub, GitLab, Bitbucket, Azure DevOps, and remote SVN in supported live flows
- a guided terminal front door
- reusable workareas
- risk and manual-step hints
- read-only verification and promotion planning
- JSON output for `plan` and `verify`

Still missing from the intended direction:

- stronger zero-config first-run polish across every provider path
- deeper remote evidence and release context in the common case
- stronger console UX for repeated daily use
- broader provider support
- richer PR, deployment, and issue evidence
