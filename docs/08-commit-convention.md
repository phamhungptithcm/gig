# Commit Convention

## Why This File Matters

If commit messages are messy, the tool becomes weak.

Good commit messages make `scan`, `find`, `diff`, and future `promote` much more reliable.

## Recommended Subject Format

Use this structure:

```text
<TICKET-ID> | <module-or-repo> | <short action>
```

Examples:

- `ABC-123 | service-a | fix login validation`
- `ABC-123 | mendix-app | update workflow`
- `ABC-123 | db | alter invoice table`

## Why This Format Works

- the ticket ID is easy to find
- the touched module is easy to see
- the action is easy to understand
- humans and tools can both read it quickly

## Minimum Rule

At minimum, every commit that should be tracked by `gig` must include a valid ticket ID in the subject line.

## Optional Footer Fields

The body may include structured footer lines such as:

```text
depends-on: XYZ-456
module: billing-service
type: fix
```

These are optional in the MVP, but they prepare the repo for later features.

## Footer Meaning

- `depends-on`: another ticket that should be checked or promoted first
- `module`: the logical area changed by this commit
- `type`: the kind of change, such as `fix`, `feature`, or `refactor`

## Default Ticket Pattern

The current default regex is:

```text
\b[A-Z][A-Z0-9]+-\d+\b
```

Examples:

- valid: `ABC-123`
- valid: `TEAM9-42`
- invalid: `abc_123`

## Team Guidance

- keep the ticket ID near the start of the subject line
- use one main ticket ID per commit subject when possible
- keep module names stable
- keep action text short and direct
- keep follow-up commits on the same ticket consistently tagged

Consistent commit messages make the tool stronger and reduce release risk.
