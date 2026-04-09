# Product Overview

## What `gig` Is

`gig` is a command-line tool for teams that release work by ticket across more than one repo.

It is useful when one ticket can pick up more commits over time before it is finally approved.

## The Simple Promise

Before someone moves a ticket from one branch to the next, `gig` helps them answer:

- what changed for this ticket?
- where is this ticket now?
- is the next move safe?
- what should be reviewed by hand first?

## Who It Helps

`gig` is useful for:

- developers who keep adding follow-up fixes to the same ticket
- QA or UAT coordinators who want to know what changed since the last review round
- release engineers who need a clear release packet and promotion plan
- outsourcing or delivery leads who need confidence across many repos

## What The Project Can Do Today

Today, `gig` can:

- scan a workspace for repos
- find ticket commits
- inspect a ticket across repos
- compare branch state for a ticket
- show environment status like `dev -> test -> prod`
- verify a promotion as `safe`, `warning`, or `blocked`
- output a read-only promotion plan in human-readable or JSON form
- generate a Markdown or JSON release packet
- load team config from a file
- check config and repo coverage with `gig doctor`

## What It Does Not Do Yet

Today, `gig` does not:

- move commits for you
- resolve conflicts
- read Jira, PR, or deployment tools yet
- build multi-ticket release bundles yet

That is by design.
The project is still focused on helping teams make better release decisions before any write action happens.

## Why This Matters

The problem is not only "find commits by ticket."

The real problem is:

- a ticket may fail review many times
- different repos may move at different speeds
- release managers can easily miss one late follow-up commit
- DB, config, or Mendix changes may need manual review

`gig` is meant to reduce those mistakes.

## Product Direction

The project is moving toward:

- better ticket visibility across repos and environments
- clearer release plans and release packets
- machine-readable output for CI and tooling
- richer evidence later from Jira, PRs, and deployments
