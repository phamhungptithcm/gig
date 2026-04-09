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

## Verify A Promotion

```bash
go run ./cmd/gig verify --ticket ABC-123 --from test --to main --path ./workspace --envs dev=dev,test=test,prod=main
```

## Build A Promotion Plan

```bash
go run ./cmd/gig plan --ticket ABC-123 --from test --to main --path ./workspace --envs dev=dev,test=test,prod=main
```

## Generate A JSON Release Manifest

```bash
go run ./cmd/gig plan --ticket ABC-123 --from test --to main --path ./workspace --envs dev=dev,test=test,prod=main --format json
```

## Notes

- `--path` can point to a single repo or a parent workspace
- `find`, `inspect`, `env status`, `diff`, `verify`, and `plan` currently require Git to be installed
- `inspect` shows ticket commits, branches, and simple risk signals
- `env status` shows whether each environment branch is present, aligned, behind, or missing
- `verify` returns a release verdict: `safe`, `warning`, or `blocked`
- `plan --format json` can be used as a first release manifest for CI or review tooling
- `diff` is read-only and does not execute cherry-pick
