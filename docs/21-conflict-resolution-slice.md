# Conflict Resolution Slice

## Goal

Ship the smallest useful write-enabled conflict workflow for `gig` without turning the product into a generic Git UI.

After this slice:

- `gig` can detect an active Git conflict session in the current repository
- `gig` can open a keyboard-first resolver that starts at the first unresolved conflict
- each conflict shows clearer provenance than generic `current` and `incoming` labels
- users can accept current, accept incoming, accept both in either order, or edit manually
- `gig` can show ticket, branch, commit, and risk context before the user applies a resolution

This slice should be Git-only.
SVN stays out of scope.

## Why This Slice Is Worth Doing

`gig` already answers:

- what ticket changes exist
- what is missing between branches
- what risk signals exist before promotion

The next real operational gap is what happens when teams actually try to move code and hit conflicts.

That is where release flow usually slows down:

- people do not know whether `current` means their branch or the branch being replayed
- people do not know which ticket or branch a side came from
- people resolve one conflict, scroll around, lose context, and miss the next one
- people choose `accept both`, but still need help knowing what to delete or double-check
- existing tools often help at file level, but not at release or ticket level

`gig` already has ticket, branch, and risk knowledge.
That makes conflict resolution a natural next slice if it stays focused.

## Real-World Signals From Existing Tools

The current market signal is clear:

- one-click `accept current`, `accept incoming`, and `accept both` is now baseline UX
- teams still ask for better provenance, safer semantics, and preflight checking
- merge editors still get called out for confusing labels and poor performance on simple conflicts

Practical gaps that show up repeatedly:

### 1. `current` and `incoming` are not enough

Users still get confused about which side they are accepting.
This gets worse in rebases because Git swaps the usual `ours` and `theirs` semantics during a rebase merge.

For `gig`, the resolver should always show:

- side label
- branch or replay source
- ticket IDs visible in the side's commit subject or body
- short commit hash and subject

Example:

```text
Current  main              commit 91ab23cd  OPS-99 tighten validation
Incoming feature/ABC-123   commit fa39187f  ABC-123 fix login flow
```

For rebase, do not only say `ours` or `theirs`.
Show the operation-aware label:

```text
Base line     main              commit 91ab23cd  OPS-99 tighten validation
Replayed pick ABC-123           commit fa39187f  ABC-123 fix login flow
```

### 2. Existing tools handle file resolution, not ticket-aware resolution

Most tools can resolve a file.
They usually do not answer:

- which branch caused this side
- which ticket does this conflicting side belong to
- whether choosing one side drops a ticket dependency or DB/config change
- whether a conflict is outside the requested ticket scope

That is the biggest differentiation opportunity for `gig`.

### 3. Teams want dry-run or preflight conflict visibility

Users still ask tools like lazygit for a merge dry-run to know whether conflicts will happen before mutating the working tree.

That suggests `gig` should not stop at an in-conflict editor.
It should also have a read-only preflight mode later.

### 4. Performance and clarity are still active pain points

VS Code merge editor has public issues describing confusing behavior and slow loading on simple conflicts.

This means the `gig` slice must avoid:

- parsing every file on every keypress
- syntax-highlighting giant buffers by default
- showing a huge full-file diff when only one conflict chunk matters
- ambiguous side labels

## Product Decision

Do not build a generic full-screen code editor.

Build a focused conflict navigator for Git operations that are already stopped on conflicts:

- `merge`
- `rebase`
- `cherry-pick`

This keeps the first slice small, fast, and aligned with the product.

It also avoids turning `gig` into a replacement for VS Code, Neovim, or JetBrains.

## Recommended Command Shape

### Phase 1

Add a new command group:

```bash
gig resolve start --path .
gig resolve status --path . --format human|json
```

`gig resolve start`:

- opens the interactive resolver only when the repo is already in a conflicted Git state
- auto-focuses the first unresolved conflict
- works file by file and conflict block by conflict block

`gig resolve status`:

- stays non-interactive
- reports operation type, branch context, unresolved file count, unsupported conflicts, and next recommended action

### Phase 2

Add preflight checking:

```bash
gig resolve check --from <branch> --to <branch> --path . [--ticket ABC-123]
```

This command stays read-only and answers:

- will this move conflict
- which files are likely to conflict
- which risks are involved before the merge or cherry-pick starts

## Interaction Model

The first useful resolver should be keyboard-first and opinionated.

### Layout

One active conflict at a time:

- top bar: repo, operation type, branch context, conflict index
- left pane: file list with unresolved counts
- main pane: current/base/incoming/result preview for the active conflict block
- footer: compact key legend

### Default Flow

When the resolver opens:

1. detect the Git operation
2. collect unresolved files
3. jump to the first unresolved conflict block
4. show the safest action options immediately
5. after applying a resolution, auto-advance to the next unresolved block

### Keyboard Plan

Recommended defaults:

- `1` accept current side
- `2` accept incoming side
- `3` accept both, current then incoming
- `4` accept both, incoming then current
- `e` edit manually in `$EDITOR`
- `n` next conflict
- `p` previous conflict
- `N` next file
- `P` previous file
- `d` details for commit, branch, and ticket provenance
- `r` risk explanation for the active conflict
- `u` undo last applied choice
- `s` stage resolved file when no conflict markers remain
- `q` quit without continuing the Git operation

Important:

- `accept both` in reverse order should be a first-class action
- this is already a requested gap in other merge UIs and is easy value for `gig`

## What Makes `gig` Better Than Generic Conflict Tools

### 1. Provenance-first labels

Do not show only:

- current
- incoming

Show:

- side name
- branch or replay source
- commit hash
- ticket ID
- short commit subject

### 2. Ticket-aware scope warnings

If the incoming side belongs to a different ticket than the one the user thinks they are promoting, show it directly.

Examples:

- `Incoming side includes OPS-99, not only ABC-123.`
- `Accept current will drop the only visible ABC-123 changes in this file.`
- `Accept incoming keeps a DB migration risk signal.`

### 3. Risk-guided combine mode

`Accept both` should not be blind.

Before applying it, `gig` should classify the active block:

- low risk: additive lines on both sides with no obvious overlap
- medium risk: duplicate imports, duplicate object keys, repeated config entries
- high risk: delete-vs-modify, schema or migration conflicts, lockfiles, generated files, binary conflicts

The UI should say what to re-check after combine.

Example:

```text
Combine risk: medium
- likely duplicate import: `useEffect`
- verify order of middleware registration
- remove one duplicate line before staging
```

### 4. Release-context hints

Reuse existing `gig` knowledge where possible:

- DB and config risk signals
- declared dependency warnings
- environment context when the conflict is part of a planned promotion flow

This is the product advantage.
Do not waste it.

## Scope Boundaries

This slice should cover:

- detection of active conflict state
- interactive resolution for text conflicts
- commit and branch provenance
- keyboard-first navigation
- risk hints for accept-current, accept-incoming, and accept-both
- safe file writes and staging
- status output and tests

This slice should not cover:

- launching a merge, rebase, or cherry-pick from `gig`
- automatic conflict-free merging across the full repo
- binary conflict editing
- full IDE editing
- SVN conflict resolution
- auto-commit or auto-push

## Architecture Plan

Keep the base `scm.Adapter` interface unchanged for now.

Use Git-only optional interfaces, similar to other capability extensions already used in the codebase.

### New Package

Create:

- `internal/conflict`

Recommended core model:

```go
type OperationType string

const (
    OperationMerge      OperationType = "merge"
    OperationRebase     OperationType = "rebase"
    OperationCherryPick OperationType = "cherry-pick"
)

type OperationState struct {
    Type           OperationType
    RepoRoot       string
    HeadBranch     string
    HeadCommit     string
    OtherCommit    string
    OtherBranch    string
    ReplayCommit   string
    ReplaySubject  string
    ReplayTicketIDs []string
}

type Side struct {
    Label       string
    Commit      string
    Branch      string
    TicketIDs   []string
    Subject     string
    Content     []byte
}

type ConflictBlock struct {
    Index     int
    Base      *Side
    Current   Side
    Incoming  Side
    Path      string
    StartLine int
    EndLine   int
}

type FileConflict struct {
    Path            string
    Kind            string
    Blocks          []ConflictBlock
    UnsupportedReason string
}
```

Exact names can change.
What matters is:

- operation-aware context
- file-level and block-level conflict objects
- side provenance
- room for risk annotations

### Optional Git Interfaces

Recommended optional interfaces:

```go
type conflictProvider interface {
    ConflictState(ctx context.Context, repoRoot string) (conflict.OperationState, error)
    ConflictFiles(ctx context.Context, repoRoot string) ([]conflict.FileConflict, error)
}

type conflictResolver interface {
    ApplyResolution(ctx context.Context, repoRoot string, path string, resolution conflict.Resolution) error
    StageResolvedFile(ctx context.Context, repoRoot string, path string) error
}
```

Git should implement them.
SVN should simply not expose them.

## Git Implementation Strategy

### Detect Operation State

Use Git metadata instead of parsing human-readable output.

Read from:

- `MERGE_HEAD`
- `CHERRY_PICK_HEAD`
- `REBASE_HEAD`
- rebase metadata under `.git/rebase-merge` or `.git/rebase-apply`
- `HEAD`

For rebase conflicts:

- show the commit currently being replayed
- show the upstream or onto branch when it can be resolved
- surface ticket IDs from the replayed commit subject or body

### Detect Conflict Files

Use stable Git plumbing:

- `git status --porcelain=v2 -z --untracked-files=no`
- `git ls-files -u -z`

That gives:

- only unresolved entries
- stage 1, 2, and 3 blob IDs
- conflict type and path

### Load Side Contents

Use the staged blobs directly:

- base via stage 1
- side A via stage 2
- side B via stage 3

Avoid reparsing the working tree file as the source of truth when possible.
The working tree may already contain partial user edits.

### Apply Resolution

For `accept current` and `accept incoming`:

- write the chosen side content into the file
- preserve file mode and line endings when possible
- stage only that file

For `accept both`:

- synthesize merged content block by block
- support both orderings:
  - current then incoming
  - incoming then current
- dedupe exact duplicate lines inside the block
- never silently dedupe near-duplicate lines

For `edit manually`:

- materialize the current result to disk
- open `$EDITOR` on the file
- on return, re-scan the file for unresolved markers before staging

### Do Not Depend On `git checkout --ours` Labels For UX

You can still use Git stage semantics internally.
But the UI must not expose raw `ours` and `theirs` terms without context.

Rebase semantics are too confusing.

## Risk Heuristics For Combine

The first slice does not need AST-aware merge intelligence.
It does need practical heuristics.

Recommended initial rules:

### Low risk

- both sides only add distinct lines
- conflict is in comments or markdown prose
- exact duplicate lines can be collapsed safely

### Medium risk

- import blocks
- JSON, YAML, or XML object sections
- route lists
- env or config files
- test snapshots

### High risk

- SQL migrations
- lockfiles
- delete-vs-modify
- rename/delete
- generated files
- files already flagged by existing `gig` risk signals

### Blocked in MVP

- binary conflicts
- submodule conflicts
- symlink conflicts
- conflicted mode changes without plain text content

## Performance Guardrails

The resolver must feel instant on real repos.

Rules:

- only fully parse the active file
- load other file metadata without reading full content
- cache blob contents by object ID
- do not syntax-highlight by default
- do not diff the entire repository view on every keypress
- keep risk analysis incremental and file-local
- avoid scanning untracked files

If a file is too large:

- show a large-file mode banner
- fall back to chunk-only preview
- disable non-essential hints

## Safety Model

This is the most important part.

`gig resolve` should be safer than raw manual editing.

Required controls:

- no automatic `git rebase --continue`, `git merge --continue`, or `git cherry-pick --continue` on first slice
- no implicit commit creation
- no branch switching
- no write when there is no active conflict state
- unsupported conflicts stay visible and clearly manual
- every applied resolution can be undone within the session
- the file is never staged if conflict markers remain

Recommended session artifact:

- `.git/gig/resolve-session.json`

Store:

- operation type
- file list
- applied choices
- timestamps
- provenance metadata

This gives auditability and makes crash recovery simpler.

## CLI And Output Plan

### `gig resolve status`

Human output should show:

- repository path
- operation type
- head branch
- replay or other branch info
- unresolved files
- unsupported conflicts
- next recommended action

JSON output should expose:

- operation type
- current branch
- other branch or replay commit
- unresolved file count
- file paths
- unsupported conflicts

### `gig resolve start`

This command is interactive.
Do not force JSON output.

If the terminal is not interactive:

- fail fast
- suggest `gig resolve status`

## Testing Plan

### Adapter Tests

Create temporary Git repos that reproduce:

- merge modify-vs-modify conflict
- rebase conflict with swapped semantics
- cherry-pick conflict
- delete-vs-modify conflict
- lockfile or config conflict

Verify:

- operation detection
- side provenance
- stage parsing
- file and block ordering

### Service Tests

Test:

- risk heuristics
- ticket extraction from conflict-side commits
- `accept both` ordering
- duplicate-line handling
- unsupported conflict classification

### CLI Tests

For non-interactive commands:

- golden tests for `resolve status`
- JSON contract tests

For interactive behavior:

- unit tests around model transitions instead of brittle raw terminal snapshots
- integration tests for apply, undo, stage, and next-conflict navigation

## Implementation Phases

### Slice A. Read-only conflict state

Ship first:

- `gig resolve status`
- Git conflict-state detection
- unresolved file listing
- operation provenance

Why:

- low risk
- useful immediately
- gives the team stable contracts before the TUI lands

### Slice B. Keyboard-first text conflict resolver

Ship next:

- `gig resolve start`
- current, incoming, both-current-first, both-incoming-first
- manual edit flow
- risk hints

### Slice C. Preflight conflict check

Ship after:

- `gig resolve check`
- dry-run conflict preview before mutating the worktree

This is the right order because it reuses the same conflict model while protecting the existing product philosophy.

## Recommendation On TUI Stack

Do not use React for this.
This is a Go CLI.

Two realistic options:

- use a small Go TUI framework such as Bubble Tea for event handling and viewport management
- build a narrower ANSI-based terminal UI by hand

Recommendation:

- use Bubble Tea for the interactive state machine
- keep rendering plain and light
- avoid ornamental styling and heavy dependencies beyond what the flow actually needs

Reason:

- faster to ship than a hand-rolled terminal event loop
- still performant if the model stays file-local and chunk-local
- easier to test as a state machine

## The Product Bet

The valuable bet is not:

- another pretty merge editor

The valuable bet is:

- a fast conflict resolver that knows branch, ticket, and release risk context

That is the gap generic tools do not cover well.

## Source Notes

Useful references that informed this plan:

- GitHub Docs: resolving merge conflicts on GitHub
  - https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/addressing-merge-conflicts/resolving-a-merge-conflict-on-github
- GitHub Docs: resolving conflicts after rebase
  - https://docs.github.com/en/get-started/using-git/resolving-merge-conflicts-after-a-git-rebase
- Git rebase docs, including swapped semantics during rebase and `--show-current-patch`
  - https://git-scm.com/docs/git-rebase
- Git status docs for stable porcelain conflict parsing
  - https://git-scm.com/docs/git-status
- Git rerere docs for reuse of recorded resolutions
  - https://git-scm.com/docs/git-rerere
- VS Code issue: merge editor as Git mergetool request
  - https://github.com/microsoft/vscode/issues/153340
- VS Code issue: merge editor reported as slow and confusing on simple conflicts
  - https://github.com/microsoft/vscode/issues/192580
- GitHub Community: one-click merge conflict resolution in the web UI
  - https://github.com/orgs/community/discussions/175270
- GitHub Community: warning about GitHub UI conflict resolution mutating the head branch
  - https://github.com/orgs/community/discussions/52762
- lazygit discussion: merge dry-run request before mutating branches
  - https://github.com/jesseduffield/lazygit/discussions/3004
