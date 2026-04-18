# Portfolio Guide

This page helps position `gig` as a strong portfolio project for recruiters, hiring managers, and GitHub visitors.

## Best Positioning Sentence

Use this when you need one concise description:

`Built a remote-first release audit CLI that reconciles ticket evidence across multiple repositories, verifies promotion readiness, and generates release-ready output for humans and automation.`

## Strong Resume Bullets

Choose the version that best fits your target role.

### Product And Platform Engineering Angle

- Built a remote-first release audit CLI that inspects GitHub, GitLab, Bitbucket, Azure DevOps, and SVN repositories directly, reducing release verification from manual repo-by-repo review to one deterministic workflow.
- Built a release audit workflow that reduced a representative 2-repo check from 6 terminal commands / 7 manual steps to 1 command / 1 step, while keeping the output deterministic and auditable.

### Developer Tooling Angle

- Designed and shipped a ticket-aware CLI for multi-repo delivery that reconciles cross-branch commit evidence, classifies release risk, and exports Markdown and JSON release packets.

### AI Systems Angle

- Layered audience-specific AI release briefings on top of deterministic source-control evidence, keeping LLM output explainable and grounded in auditable repository state.

## What Makes `gig` Look Enterprise-Grade

- clear product problem tied to release governance and delivery risk
- multi-provider repository support instead of one narrow integration
- strong deterministic core before AI augmentation
- human-readable and machine-readable outputs
- explicit product direction toward zero-config, remote-first workflows

## Best Talking Points For Interviews

- why ticket reconciliation is harder than raw commit search
- how remote-first access changes onboarding and daily usage
- why deterministic evidence should stay below any AI summarization layer
- how workareas reduce repeated setup across many clients or products
- how terminal UX can be summary-first without hiding critical detail

## How To Make The Repo Star-Worthy

The repo earns stars when the value is obvious in under a minute.

Do these consistently:

- keep the README focused on `inspect`, `verify`, and `manifest`
- keep a short terminal demo near the top of the repo
- show one clear enterprise use case instead of every possible command
- keep the roadmap credible and incremental
- make installation and first-run steps frictionless

## Best Assets To Share

- the README thumbnail linking to the demo guide
- a 45 to 60 second terminal clip
- one screenshot each for inspect, verify, and AI brief
- a short architecture or product narrative in the docs site
- one benchmark snapshot that shows workflow compression honestly, not just raw millisecond claims

Recommended benchmark asset:

- `docs/assets/release-audit-benchmark.svg` for README embeds, portfolio case studies, and social previews

## Suggested Portfolio Stack

If you are presenting `gig` on a portfolio site or in a case study, keep the structure simple:

1. problem: release teams miss follow-up fixes across repos
2. solution: `gig` reconciles ticket evidence and verifies readiness
3. proof: terminal demo plus deterministic output screenshots
4. architecture: remote-first provider access, workareas, deterministic audit core, optional AI layer
5. outcome: easier release decisions, clearer audit trails, better developer experience
