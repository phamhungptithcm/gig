# Projects

A project is saved context for repeated `gig` work. It remembers repo scope and branch defaults so commands stay short outside a checkout.

`workarea` is the older command name and still works as an alias.

## When To Save A Project

Use one when:

- the team checks the same repo or release path often
- `gig` cannot infer branches from provider metadata
- you want short commands from outside the repo checkout
- release-day commands should be repeatable in CI or shared runbooks

Do not create a project just to try `gig`. From inside a supported Git checkout, start with `gig ABC-123`.

## Save A Remote Project

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
```

Then run:

```bash
gig ABC-123
gig verify ABC-123
gig packet ABC-123
```

## Save A Local Project

```bash
gig project add local-payments --path . --from staging --to main --use
```

Use this for local Git/SVN fallback when branch topology is not inferable.

## Switch Projects

```bash
gig project list
gig project use payments
gig project show
```

If you run `gig` without arguments, saved projects appear in the guided front door.

## Project Rules

- Explicit flags win over project defaults.
- The current checkout remote wins over the global current project unless you pass `--project`.
- A project can store a remote repo target or local path.
- `--from`, `--to`, and `--envs` are useful when provider topology is ambiguous.
- Projects reduce command length; they are not required for first use.

## Common Failures

| Failure | Next action |
| --- | --- |
| Wrong current project | Run `gig project list`, then `gig project use <name>`. |
| Branch defaults are wrong | Re-add or update the project with correct `--from` and `--to`. |
| Provider auth fails | Run `gig login` for the project provider. |
| Local path moved | Recreate the project with the new `--path`. |
