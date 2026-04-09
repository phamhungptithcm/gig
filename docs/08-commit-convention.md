# Commit Convention

## Why This Matters

`gig` becomes much more useful when commit messages are consistent.

Good commit subjects make it easier for:

- `gig find`
- `gig inspect`
- `gig diff`
- `gig verify`
- `gig plan`

## Recommended Subject Format

Use this simple format:

```text
<TICKET-ID> | <module-or-repo> | <short action>
```

Examples:

- `ABC-123 | service-a | fix login validation`
- `ABC-123 | web-ui | adjust summary screen`
- `ABC-123 | db | add invoice column`
- `ABC-123 | mendix-app | update workflow`

## Why This Format Works

- the ticket ID is easy to spot
- the touched area is easy to understand
- the action is short and readable
- both humans and tools can scan it quickly

## Minimum Rule

At minimum, every tracked commit should include a valid ticket ID in the subject line.

## Optional Footer Fields

The body can include extra structured lines such as:

```text
depends-on: XYZ-456
module: billing-service
type: fix
```

These are optional today, but useful for later dependency and release-packet features.

## Team Guidance

- keep the ticket ID near the start of the subject
- keep module names stable
- keep the action short and direct
- keep follow-up commits on the same ticket consistently tagged

Consistent commit messages reduce release mistakes.
