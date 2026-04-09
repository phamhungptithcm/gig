# Architecture

## Design Goal

Keep the code simple, testable, and easy to extend.

The main rule is:
the CLI should not contain business logic.

## Layers

### CLI Layer

Role:
Receive user input and map it to application actions.

Responsibilities:

- parse commands and flags
- validate required arguments at command level
- call services
- return exit codes
- route results to output renderers

Main package:

- `cmd/gig`
- `internal/cli`

### Service Layer

Role:
Run use cases.

Responsibilities:

- scan workspaces
- search commits by ticket
- compare branches
- later build promotion plans

Main packages:

- `internal/repo`
- `internal/ticket`
- `internal/diff`
- future `internal/promote`

### Core Domain Layer

Role:
Define the shared language of the tool.

Responsibilities:

- repository model
- commit model
- branch diff model
- promotion plan model
- ticket change set model

Main location today:

- `internal/scm`

This can be split further later if the domain grows.

### Adapter Layer

Role:
Talk to external systems.

Responsibilities:

- Git operations
- future SVN operations
- future Jira integration

Main packages:

- `internal/scm/git`
- `internal/scm/svn`

## Request Flow

1. CLI parses command and flags.
2. Scanner resolves the input path into one or more repositories.
3. Service selects the correct SCM adapter for each repository.
4. Adapter executes SCM-specific queries.
5. Output renderer formats the service results for the terminal.

## Why This Design Works

- each layer has one clear job
- Git details do not leak into the CLI
- future SVN support can reuse the same service flow
- output format can change without rewriting business logic
