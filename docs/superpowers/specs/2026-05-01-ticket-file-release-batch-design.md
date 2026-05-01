# Ticket File Release Batch Design

Date: 2026-05-01

## Problem

`gig` currently works well when the user starts from one ticket such as
`ABC-123`. Real release work usually audits many tickets together. The product
already supports `--ticket-file`, but the user experience does not clearly teach
people what that file should contain, how to create a sample, or when to prefer
batch commands over typing one ticket at a time.

This design makes `--ticket-file` the primary release batch input for one
repository and one promotion path.

## Goals

- Keep existing one-ticket commands working.
- Keep existing plain text `--ticket-file` files working.
- Add a standard CSV ticket file shape that spreadsheet users can edit easily.
- Let users generate a sample ticket file from the CLI.
- Surface batch release commands in smart suggestions and front-door help.
- Keep the first usable slice scoped to one repository and one source-to-target
  promotion path.
- Preserve remote-first, zero-config-first behavior.
- Work consistently on macOS and Windows.

## Non-Goals

- Do not add per-row repository, source branch, or target branch support in this
  slice.
- Do not introduce a new `--release-file` flag.
- Do not require `gig.yaml` to use release batch verification.
- Do not claim multi-repository release CSV support until a later vertical slice
  wires it end to end.

## User Workflow

The recommended flow is:

```bash
gig ticket-file sample --out tickets.csv
gig verify --ticket-file tickets.csv --repo github:owner/name --from staging --to main --out release-audit.xlsx
gig packet --ticket-file tickets.csv --repo github:owner/name --from staging --to main --out release-packet.xlsx
```

When a project remembers the repo and branch path, commands become shorter:

```bash
gig project add payments --repo github:owner/name --from staging --to main --use
gig ticket-file sample --out tickets.csv
gig verify --ticket-file tickets.csv --out release-audit.xlsx
gig packet --ticket-file tickets.csv --out release-packet.xlsx
```

## Ticket File Format

Plain text remains supported:

```txt
# release tickets
ABC-123
XYZ-456
```

CSV becomes the standard spreadsheet-friendly format:

```csv
ticket,summary,owner,notes
ABC-123,Login validation,Backend Team,QA approved on dev
XYZ-456,Checkout fix,Frontend Team,Needs smoke test
```

Rules:

- `ticket` is required.
- `summary`, `owner`, and `notes` are optional metadata.
- Header names are case-insensitive.
- UTF-8 BOM and CRLF line endings are accepted.
- Blank rows are ignored.
- Duplicate tickets are de-duplicated by normalized ticket ID.
- Formula-like cell values in generated exports continue to be escaped by the
  existing export layer.

## CLI Surface

Add:

```bash
gig ticket-file sample --out tickets.csv
gig ticket-file sample --out tickets.csv --force
```

Behavior:

- `--out` is required.
- The sample command writes CSV in this slice and recommends a `.csv` path.
- Existing files are not overwritten unless `--force` is provided.
- The generated sample uses stable example IDs such as `ABC-123` and `XYZ-456`.
  Validation still uses the configured ticket pattern when commands read the
  completed file.

Keep the existing command flags:

```bash
gig plan --ticket-file tickets.csv ...
gig verify --ticket-file tickets.csv ...
gig packet --ticket-file tickets.csv ...
gig assist release --release rel-2026-04-09 --ticket-file tickets.csv ...
```

## Parser Design

Replace the current line-only ticket-file reader with a small parser in
`internal/cli` or a dedicated package if the logic grows. The parser returns:

- normalized ticket IDs for existing plan, verify, packet, and assist flows
- optional metadata records for future output enrichment
- row-aware validation errors

The first implementation can keep command services unchanged by passing only the
ticket ID slice. Metadata can be parsed and preserved behind an internal type so
future export enrichment does not require another file format migration.

Error examples:

```text
ticket file tickets.csv row 4: missing ticket
ticket file tickets.csv row 7: invalid ticket "abc"
ticket file tickets.csv: missing required column "ticket"
```

## Smart Suggestions

Front-door and post-command suggestions should teach the batch path:

```text
sample   gig ticket-file sample --out tickets.csv
audit    gig verify --ticket-file tickets.csv --out release-audit.xlsx
packet   gig packet --ticket-file tickets.csv --out release-packet.xlsx
```

When the current project has repo and branch defaults, omit `--repo`, `--from`,
and `--to`. When no project is saved but a remote checkout is detected, include
the inferred remote scope. When topology is ambiguous, keep the existing
explicit branch suggestion and show the batch command with `<source>` and
`<target>` placeholders.

Prompt help should mention that `verify --ticket-file tickets.csv` and
`packet --ticket-file tickets.csv` are preferred for release batches.

## Output And Export

No output contract change is required for the first slice. Existing batch
renderers and XLSX/CSV export builders already accept multiple plans,
verifications, or packets.

Recommended output commands:

```bash
gig verify --ticket-file tickets.csv --out release-audit.xlsx
gig verify --ticket-file tickets.csv --out release-audit.csv
gig packet --ticket-file tickets.csv --out release-packet.xlsx
gig packet --ticket-file tickets.csv --format csv --out release-packet/
```

Later slices can add ticket-file metadata columns to the Summary, Scope, or
Metadata sheets after the deterministic command path is stable.

## Cross-Platform Behavior

Use Go standard library file and CSV handling:

- `os.OpenFile` with exclusive-create semantics for safe sample generation.
- `encoding/csv` for CSV parsing.
- `filepath` for display and path handling.
- no shell-dependent commands.
- accept Windows CRLF line endings.
- avoid permissions or symlink behavior beyond normal file creation.

This keeps behavior smooth on macOS, Windows, and CI.

## Testing

Add focused tests for:

- plain text ticket files still work
- CSV ticket files with header work
- UTF-8 BOM and CRLF are accepted
- duplicate tickets are de-duplicated
- missing `ticket` column errors clearly
- invalid ticket errors include row number
- `gig ticket-file sample --out tickets.csv` creates expected CSV
- `--force` overwrites and default mode refuses overwrite
- smart suggestions include sample, audit, and packet batch commands
- front-door parser still passes through `--ticket-file`

Run focused tests first:

```bash
go test ./internal/cli
```

Then run:

```bash
go test ./...
```

If docs or user-facing help change, also run:

```bash
./.venv-docs/bin/mkdocs build --strict
```

## Rollout

Implement as one vertical slice:

1. Add sample command and parser support.
2. Wire parser into existing `resolveTicketIDs` flow.
3. Add smart suggestion and help text.
4. Update README, CLI spec, release-day workflow, and command reference.
5. Add tests and intentional golden updates.

This preserves existing local and remote release behavior while making the
multi-ticket release path obvious.
