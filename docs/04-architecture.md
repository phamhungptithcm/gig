# Architecture

## Design Goal

Keep the code simple, testable, and easy to grow.

The main rule is:
the CLI should stay thin.

## Main Layers

### CLI Layer

This layer turns user input into application calls.

Responsibilities:

- parse commands and flags
- show help and usage
- normalize simple input such as ticket ID or path
- call the right service
- choose human or JSON output
- return exit codes

Main packages:

- `cmd/gig`
- `internal/cli`

### Service Layer

This layer contains the real use cases.

Responsibilities:

- scan workspaces
- find ticket commits
- compare branches
- inspect a ticket across repos
- evaluate environment status
- build read-only verification and promotion plans

Main packages:

- `internal/repo`
- `internal/ticket`
- `internal/diff`
- `internal/inspect`
- `internal/plan`

### Domain Layer

This layer holds the shared shapes the rest of the code works with.

Examples:

- repository info
- commit refs
- branch comparison results
- risk signals
- environment status
- promotion plan and verification results

Main locations today:

- `internal/scm`
- `internal/inspect`
- `internal/plan`

### Adapter Layer

This layer talks to external tools such as Git.

Responsibilities:

- detect repos
- resolve branch names
- search commits
- compare branches
- inspect files changed by a commit

Main packages:

- `internal/scm/git`
- `internal/scm/svn`

## Request Flow

1. CLI reads the command and flags.
2. The scanner turns the path into one or more repositories.
3. The service chooses the correct adapter for each repo.
4. The adapter runs SCM-specific queries.
5. The output package renders the result for terminal or JSON.

## Why This Design Works

- each layer has one clear job
- Git details do not leak into the CLI
- new output formats can reuse the same service results
- future integrations can be added without rewriting the commands
