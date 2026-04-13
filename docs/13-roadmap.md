# Roadmap

## Direction In One Sentence

`gig` is moving from a local, config-heavy ticket helper into a remote-first release audit CLI with zero-config onboarding and project workareas.

## What Is Already Here

The current codebase already has useful release-analysis foundations:

- ticket inspection across repositories
- branch comparison and promotion verification
- risk signals for DB, config, and Mendix-style changes
- Markdown and JSON release packets
- GitHub-backed remote inspection in the current live flow
- local workspace scanning and config overrides
- read-only safety by default

## Product Priorities

The next roadmap should follow this order.

### Phase 1. Zero-Config First Run

Goal:
install `gig`, run `gig`, and get to first useful result without a setup document.

Priority work:

- guided provider login
- GitHub-first remote repository connection
- protected-branch and release-flow auto-detection
- default ticket search without repeated `--from`, `--to`, or `--config`
- a simpler first-run command surface

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

- named workareas per client, product, or release stream
- saved repo scope, provider session, branch topology, and output preferences
- project picker and recent-workarea flow
- keyboard-first console layout with progressive detail
- readable audit views instead of raw walls of commits

### Phase 4. Team Memory And Release Evidence

Goal:
help teams use `gig` as the audit layer before promotion.

Priority work:

- richer release packets and reusable audit bundles
- optional project metadata and team notes
- deployment, build, and issue-tracker evidence
- stronger JSON contracts for downstream tooling
- release-level views that aggregate many ticket audits cleanly

### Phase 5. Controlled Actions And Broader Coverage

Goal:
expand provider coverage and add carefully guarded write actions only after trust is earned.

Priority work:

- GitLab and Bitbucket remote support
- remote SVN and enterprise edge cases where they still matter
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
