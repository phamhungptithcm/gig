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

## Compare Promotion Readiness

```bash
go run ./cmd/gig diff --ticket ABC-123 --from dev --to test --path ./workspace
```

## Notes

- `--path` can point to a single repo or a parent workspace
- `find` and `diff` currently require Git to be installed
- `diff` is read-only and does not execute cherry-pick
