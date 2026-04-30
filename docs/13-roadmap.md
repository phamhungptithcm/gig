# Roadmap

## Direction In One Sentence

`gig` is moving from a capable local release helper into a remote-first, zero-config-first release audit CLI.

## What Is Already Shipping

The current codebase already delivers:

- ticket-aware commit discovery across repositories
- release verification and risk hints
- Markdown and JSON release packets
- remote inspection for GitHub, GitLab, Bitbucket, Azure DevOps, and remote SVN
- current-checkout remote inference from Git `origin`
- high-confidence protected-branch inference with explicit fallback when topology is ambiguous
- local Git and SVN fallback flows
- reusable projects, with `workarea` kept as a compatibility alias
- a guided `gig` front door
- optional DeerFlow-backed ticket, release, and conflict briefings with saved follow-up sessions and richer release evidence
- structured diagnostics for auth and topology support traces

## What Comes Next

### 1. Sharper First-Run Experience

Priority:

- reduce friction from install to first useful audit
- let users run `gig ABC-123`, `gig verify ABC-123`, and `gig packet ABC-123` from inside the repo without typing `--repo`
- improve repository discovery and project reuse

### 2. Stronger Remote Audit Depth

Priority:

- improve cross-branch and cross-repo ticket evidence
- strengthen follow-up fix detection
- make `safe`, `warning`, and `blocked` verdicts easier to trust at a glance
- raise more providers from basic release evidence to deep release evidence

### 3. Better Project And Console UX

Priority:

- cleaner multi-project switching
- stronger summary-first terminal output
- faster keyboard-driven navigation for repeated use

### 4. Better Release Evidence

Priority:

- richer release packets and JSON contracts
- stronger audit bundles for QA, release, and client-facing stakeholders
- keep deep release evidence strong across GitHub, GitLab, and Azure DevOps while closing the remaining Bitbucket parity gap
- optional AI explanations that stay grounded in deterministic evidence

### 5. Controlled Expansion

Priority:

- fill the remaining provider and enterprise gaps without making the product config-heavy again
- add guarded write actions only after the read-only audit path is strong enough

## What Should Not Lead

These are useful, but they should not become the front door:

- config-first onboarding
- local-workspace-first storytelling
- exposing engine internals before user intent is clear
- adding more required branch flags where source control can answer directly

## Product Rule

Safe release decisions matter more than clever automation.
