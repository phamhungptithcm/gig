# AI Assist

AI assist is optional.
It explains deterministic `gig` evidence for a specific audience.

Use it only after the normal audit path works:

```bash
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

## Check Readiness

```bash
gig assist doctor
gig assist setup
```

Use `doctor` to check readiness.
Use `setup` to bootstrap the bundled DeerFlow sidecar config.

## Ticket Brief

```bash
gig explain ABC-123 --audience qa
gig explain ABC-123 --audience client
gig explain ABC-123 --audience release-manager
```

## Follow-Up Questions

```bash
gig ask "what is still blocked?"
gig ask "what changed since the last brief?"
gig resume
```

`gig ask` resumes the saved session for the current project, repo target, or workspace.
Before answering, `gig` refreshes the deterministic bundle.

## Release Brief

```bash
gig assist release --release rel-2026-04-09 --audience release-manager
```

or:

```bash
gig assist release --ticket-file tickets.txt --repo github:owner/name --audience release-manager
```

## Rules

- The model explains `gig` evidence; it should not invent release facts.
- `gig` remains the source of truth.
- Use deterministic `verify` and `packet` output for final release decisions.
- Keep AI assist out of the first-run path unless the user explicitly wants a narrative brief.
