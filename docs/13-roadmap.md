# Roadmap

## Direction In One Sentence

`gig` is moving from a local, config-heavy ticket helper into a remote-first release audit CLI with zero-config onboarding and project workareas.

## What Is Already Here

The current codebase already has useful release-analysis foundations:

- public npm distribution for one-command install and upgrade
- ticket inspection across repositories
- branch comparison and promotion verification
- risk signals for DB, config, and Mendix-style changes
- Markdown and JSON release packets
- GitHub, GitLab, Bitbucket, and Azure DevOps-backed remote inspection in the current live flow
- remote SVN-backed inspection in the current live flow
- initial workarea save and switch commands
- an initial guided front door when users run `gig` without a subcommand
- an experimental DeerFlow-backed assist path that consumes `gig` ticket, release, and conflict bundles
- a local `gig assist doctor` readiness check for the bundled DeerFlow sidecar
- a local `gig assist setup` bootstrap command for the bundled DeerFlow sidecar
- local workspace scanning and config overrides
- read-only safety by default

## Product Priorities

The next roadmap should follow this order.

### Phase 1. Zero-Config First Run

Goal:
install `gig`, run `gig`, and get to first useful result without a setup document.

Priority work:

- guided provider login
- GitHub, GitLab, Bitbucket, and Azure DevOps remote repository connection
- protected-branch and release-flow auto-detection
- default ticket search without repeated `--from`, `--to`, or `--config`
- a simpler first-run command surface on top of the initial guided dashboard

### Phase 2. Remote Audit Core

Goal:
make remote ticket audit stronger than local repo scanning for the common case.

Priority work:

- online branch search for ticket evidence
- cross-repo remote inspection from provider APIs
- PR or merge-request evidence
- dependency and follow-up fix detection across connected repos
- safer audit output with clearer `safe`, `warning`, and `blocked` reasoning

### Phase 3. Workareas And Console UX

Goal:
support people who work across many projects and need `gig` to remember context.

Priority work:

- richer workareas per client, product, or release stream on top of the initial saved-workarea slice
- broader saved defaults and cleaner project switching
- keyboard-first search, recent-history ranking, and richer project browsing
- keyboard-first console layout with progressive detail
- readable audit views instead of raw walls of commits

### Phase 4. Team Memory And Release Evidence

Goal:
help teams use `gig` as the audit layer before promotion.

Priority work:

- richer release packets and reusable audit bundles
- optional AI briefings that explain the same deterministic ticket, release, or conflict bundle for QA, client, and release-manager audiences
- optional project metadata and team notes
- build and issue-tracker evidence
- stronger JSON contracts for downstream tooling
- release-level views that aggregate many ticket audits cleanly
- project-specific skill packs that keep AI agents aligned with `gig` product guardrails

### Phase 5. Controlled Actions And Broader Coverage

Goal:
expand provider coverage and add carefully guarded write actions only after trust is earned.

Priority work:

- enterprise edge cases that still matter after remote SVN support lands
- controlled promote helpers with explicit confirmation
- rollback guidance and richer operational reporting

## What Should Not Lead The Roadmap

These are still useful, but they should not define the front door:

- local workspace scanning as the default story
- config-first onboarding
- command growth that exposes engine internals instead of user intent
- enterprise adapter breadth before first-run usability is strong

## Product Rule

Safe release work comes before clever automation.
