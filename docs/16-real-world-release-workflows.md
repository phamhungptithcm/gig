# Real-World Release Workflows

## Why This Document Exists

The original MVP explains the problem in a simple way.

This document captures the more realistic workflow that many outsourcing and enterprise teams actually live with:

- one ticket touches many repositories
- one ticket fails verification multiple times
- one ticket keeps accumulating commits across environments
- one ticket may depend on another ticket or a manual step

If `gig` cannot model that reality, it will stay interesting but not operationally useful.

## Common Team Workflow

Many teams manage work in Jira, Asana, or a similar tracker.
Each ticket has an ID.
Developers use that ticket ID in branch names, commit messages, pull requests, and release notes.

Typical branch and environment flow:

1. a developer creates a working branch from `dev` or `staging`
2. the change is merged into the shared development branch and deployed to the dev environment
3. after dev verification, the change is merged into a test branch and deployed to QA
4. after QA passes, the same ticket may be reviewed by the client in UAT
5. only after final approval are the right changes promoted to the production line

In practice, the ticket rarely passes on the first try.

## Real Failure Loop

One ticket may need:

- Java service changes
- Python batch changes
- SQL migration scripts
- Angular UI changes
- Mendix workflow changes

Those changes may live in different repositories and move through the same environments at different speeds.

The real loop often looks like this:

1. developer implements the first version
2. QA or developer verification fails
3. more commits are added
4. the ticket is redeployed to dev or test
5. QA passes, but client review fails
6. more commits are added again
7. the cycle repeats until the ticket is finally approved

By the time the ticket is ready for production, it may have:

- many commits in the same repository
- commits across several repositories
- follow-up fixes created days after the original work
- dependencies on other tickets
- manual release steps such as DB execution or config changes

## What Usually Goes Wrong

The common release failure is not "we forgot the ticket exists."

The common failure is:

- one repository was missed
- one late follow-up commit was missed
- a database script was not promoted with the app change
- one dependent ticket was not included
- a Mendix or config change needed manual review but nobody flagged it
- branch history said the commit existed somewhere, but nobody knew whether it was actually deployed to the intended environment

## Product Implications For `gig`

This workflow means `gig` should not stop at commit search.

It should answer operational questions such as:

- where did this ticket change code or artifacts?
- which commits belong to the ticket in each repository?
- what changed after the last QA or client rejection?
- what is present in source but still missing in the target release line?
- which items are safe, risky, blocked, or manual-only?
- which dependencies or companion tickets must travel together?
- what release packet should the team review before promotion?

## Key Insight

Branch history is useful, but branch history is not enough.

For real team workflows, `gig` needs to model all of these:

- ticket change set
- environment state
- dependency edges
- risk signals
- manual steps
- release manifest

That is the difference between a useful engineering helper and a release coordination tool that teams can trust.
