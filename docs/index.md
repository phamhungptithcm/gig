# gig

`gig` is a release audit CLI that checks ticket evidence across source control and tells teams whether a change is ready to promote.

The core question is:

`Did we miss any change for this ticket?`

## Fastest Useful Path

```bash
gig
gig login
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

Use the guided `gig` front door when you are not sure what to run.
Use explicit `--repo` targets when you are outside the checkout.

## Three Daily Workflows

### Inspect One Ticket

```bash
gig ABC-123
```

Use this when you need the full ticket story: commits, branches, PRs or merge requests, linked evidence, and risk hints.

### Verify Release Readiness

```bash
gig verify ABC-123
```

Use this when you need a `safe`, `warning`, or `blocked` verdict for the next promotion.
`gig` tries to infer the promotion path from provider branch metadata.
If it cannot infer safely, it stops and tells you what flag or project setting is missing.

### Generate A Release Packet

```bash
gig packet ABC-123
```

Use this when QA, release managers, or client-facing stakeholders need a clean Markdown or JSON packet instead of raw terminal evidence.

## When To Use gig

Use `gig` when:

- one ticket may touch multiple repositories or source-control systems
- QA or client review produced follow-up fixes
- release readiness depends on what actually reached the target branch
- you need repeatable evidence for a ticket or release
- you want terminal output, JSON, and release packets from the same source of truth

Do not use `gig` as a replacement for code review, CI/CD, or human release approval.
It sits between source-control history and release decisions.

## Defaults

- Config is optional. Start without `gig.yaml`.
- Remote repositories are the preferred first path.
- Local Git/SVN mode is a fallback when remote access is unavailable or incomplete.
- Projects are optional saved context for teams that run the same repo/release checks every day.
- AI assist is optional and must stay grounded in deterministic `gig` evidence.

## If gig Cannot Infer Something

| Problem | First action |
| --- | --- |
| Not logged in | Run `gig login` and choose the provider. |
| Missing provider CLI | Install the CLI named in the error, then rerun `gig login`. |
| Unknown repo | Add `--repo github:owner/name` or choose a repo in the front door. |
| Ambiguous branches | Let `gig` ask in an interactive terminal, add `--from`/`--to`, or save a project with branch defaults. |
| Local mode needs topology | Use the interactive prompt, `--from staging --to main`, or `gig project add local --path . --from staging --to main --use`. |
| Team-specific branch model | Add `--envs` or optional config only after inference needs help. |

## Next

- [Quick Start](19-quickstart.md)
- [CLI Guide](03-cli-spec.md)
- [Troubleshooting](troubleshooting.md)
