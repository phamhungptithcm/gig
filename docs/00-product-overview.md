# Product Overview

## What `gig` Is

`gig` is being repositioned as a remote-first release audit CLI for teams that move work by ticket across many repositories.

The core question stays the same:

`Did we miss any change for this ticket?`

That question becomes hard when one ticket keeps picking up more commits across backend, frontend, database, scripts, and low-code assets before it is finally approved.

## The Simple Promise

Before someone moves a ticket or release forward, `gig` should help them answer:

- what changed for this ticket?
- what is still missing?
- what looks risky?
- what should be reviewed by hand before approval?

## Product Direction

The intended direction is:

- login once to a remote provider and inspect live repository state
- auto-detect branch topology, ticket evidence, and likely release paths
- let users create workareas for each project so setup is remembered
- keep config optional and only use it to improve quality or override detection
- make the console UX clean enough that users can stay inside `gig` while working across many repos

## Who It Should Help Most

`gig` should be strongest for:

- developers following one ticket across many repos
- QA or UAT coordinators checking what changed since the last review cycle
- release engineers who need a fast ticket audit before promotion
- delivery leads who need confidence without manually reading every repo again

## What The Current Build Already Does Well

Today, the project already has useful release logic:

- ticket inspection across repos
- branch-to-branch verification
- risk hints for DB, config, and Mendix-style changes
- Markdown and JSON release packets
- GitHub-backed remote inspection for selected flows
- local workspace scanning and optional config overrides

## Installation

Package managers publish `gig` as `gig-cli` to avoid name collisions, but the command you run after install stays `gig`.

For macOS and Linux with Homebrew:

```bash
brew tap phamhungptithcm/gig https://github.com/phamhungptithcm/gig
brew install phamhungptithcm/gig/gig-cli
gig version
```

For Windows with Scoop:

```powershell
scoop bucket add gig https://github.com/phamhungptithcm/gig
scoop install gig/gig-cli
gig version
```

For macOS and Linux without Homebrew:

```bash
curl -fsSL https://raw.githubusercontent.com/phamhungptithcm/gig/main/scripts/install.sh | sh
gig version
```

Update the installed binary:

```bash
gig update
```

## What It Does Not Do Yet

Today, `gig` does not:

- give a zero-config first-run experience across all providers
- remember projects as workareas yet
- provide the full keyboard-first console UX the product needs
- gather rich PR, deployment, and issue evidence yet
- move commits or deploy anything for you

Those are product-direction gaps, not a change in purpose.
The release-audit problem is still the center.

## Why This Matters

The problem is not only "find commits by ticket."

The real problem is:

- a ticket may fail review many times
- different repos may move at different speeds
- release managers can easily miss one late follow-up commit
- DB, config, or Mendix changes may need manual review

`gig` is meant to reduce those mistakes.

## What Success Should Look Like

After install, users should be able to:

1. run `gig`
2. log in once if needed
3. pick or create a workarea for one project
4. search a ticket online across remote branches
5. understand risk and missing changes without wiring config first
