# Examples

## Typical Workspace

```text
workspace/
├── service-a/.git
├── service-b/.git
└── ui/app/.git
```

## Scan

```bash
go run ./cmd/gig scan --path ./workspace
```

## Find A Ticket

```bash
go run ./cmd/gig find ABC-123 --path ./workspace
```

## Inspect A Ticket

```bash
go run ./cmd/gig inspect ABC-123 --path ./workspace
```

## Check Environment Status

```bash
go run ./cmd/gig env status ABC-123 --path ./workspace --envs dev=dev,test=test,prod=main
```

## Compare Promotion Readiness

```bash
go run ./cmd/gig diff --ticket ABC-123 --from dev --to test --path ./workspace
```

## Notes

- `--path` can point to a single repo or a parent workspace
- `find`, `inspect`, `env status`, and `diff` currently require Git to be installed
- `inspect` shows ticket commits, branches, and simple risk signals
- `env status` shows whether each environment branch is present, aligned, behind, or missing
- `diff` is read-only and does not execute cherry-pick
