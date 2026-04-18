# Dependency Parsing And Missing-Dependency Risk

## Goal

Ship the smallest useful dependency-aware slice for `gig` without changing the product's read-only safety model.

After this slice:

- `gig` can read declared ticket dependencies from commit trailers
- `gig inspect` can show dependency evidence for the requested ticket
- `gig plan`, `gig verify`, and `gig manifest` can flag when a declared dependency is not yet present in the target branch
- dependency findings appear in both human-readable output and structured output

## Why This Slice Comes Next

The current MVP already covers the core read-only workflow:

- repository scan
- ticket inspection
- environment status
- verification
- read-only promotion planning
- release packet generation

The next product gap is not another base command.
The next gap is better release reasoning.

The clearest missing capability is:

- understand declared dependency tickets
- prevent a single-ticket promotion from looking safe when a companion ticket is still missing

## Scope

This slice should cover:

- Git commit trailer parsing for dependency metadata
- ticket-level dependency resolution across the scanned workspace
- a new missing-dependency risk signal
- output updates for inspect, plan, verify, and manifest
- tests for parser behavior, workspace resolution, service behavior, and CLI output

This slice should not cover:

- Jira integration
- PR or deployment evidence
- multi-ticket bundle planning
- write execution such as `gig promote`
- dependency inference from file names or issue tracker links

## Source Of Truth For Dependencies

This slice should use commit trailers as the only supported dependency source.

Initial supported trailer key:

- `Depends-On`

The parser should accept case-insensitive forms such as:

- `Depends-On: XYZ-456`
- `depends-on: XYZ-456`

Recommended support in the first slice:

- repeated trailers on separate lines
- comma-separated values in one trailer line
- deduplication after normalization

Out of scope for the first slice:

- free-form dependency text in subjects
- PR body parsing
- Jira issue link parsing

## Behavior To Ship

### 1. Parse Declared Dependencies

When a commit for ticket `ABC-123` contains:

```text
Depends-On: XYZ-456
Depends-On: OPS-99, UI-7
```

`gig` should capture those dependency tickets as structured evidence for `ABC-123`.

### 2. Resolve Dependency Status Against The Release Move

For each declared dependency ticket:

1. search the scanned workspace for that dependency ticket in the source branch
2. search the scanned workspace for that dependency ticket in the target branch
3. derive a dependency status from those results

Recommended initial statuses:

- `satisfied`
  - dependency ticket is already present in the target branch
- `missing-target`
  - dependency ticket is present in the source branch but not in the target branch
- `unresolved`
  - dependency trailer exists, but the dependency ticket could not be confirmed in either source or target inside the scanned workspace

### 3. Emit Risk From Confirmed Missing Dependencies

This slice should add a new risk signal:

- code: `missing-dependency`
- level: `blocked`

Emit it when:

- a ticket declares a dependency
- the dependency is visible in the scanned workspace
- the dependency is present in the source branch
- the dependency is not present in the target branch

This is the strongest fact pattern available in the current read-only model.
It means the current promotion candidate is likely incomplete.

### 4. Emit A Softer Signal For Unresolved Dependencies

To avoid silently ignoring dependency trailers, this slice should also emit a softer signal when the workspace cannot confirm the dependency:

- code: `unresolved-dependency`
- level: `warning`

Emit it when:

- a dependency trailer exists
- the dependency is not confirmed in the target branch
- the workspace also cannot confirm it in the source branch

This keeps the tool useful in partial workspaces without pretending certainty.

## Proposed Domain Additions

The recommended design is to keep the core `scm.Adapter` interface unchanged and use optional provider interfaces, similar to how changed files are loaded today.

### New Package

Create a dedicated package:

- `internal/dependency`

Recommended shapes:

```go
type DeclaredDependency struct {
	TicketID    string
	DependsOn   string
	CommitHash  string
	CommitSubject string
	TrailerKey  string
}

type Status string

const (
	StatusSatisfied     Status = "satisfied"
	StatusMissingTarget Status = "missing-target"
	StatusUnresolved    Status = "unresolved"
)

type Resolution struct {
	TicketID        string
	DependsOn       string
	Status          Status
	FoundInSource   bool
	FoundInTarget   bool
	EvidenceCommits []string
}
```

Exact names can change during implementation, but the model needs:

- the current ticket
- the dependency ticket
- the commit evidence that declared it
- source and target presence
- a derived status that output code can render directly

## Evaluation Algorithm

The first implementation should follow this order:

1. inspect the requested ticket and collect matching commits per repository
2. load raw commit messages for those commits from adapters that support it
3. parse dependency trailers from the commit bodies
4. normalize and dedupe dependency ticket IDs
5. for each dependency ticket, search across the scanned workspace in:
   - `fromBranch`
   - `toBranch`
6. derive dependency status
7. attach the dependency resolutions to inspect and planning results
8. emit risk signals, actions, notes, and manual steps from the derived status

Important rule:

- dependency resolution is workspace-wide, not limited to the same repository as the declaring commit

That matches the real workflow where one app ticket may depend on a DB ticket in another repo.

## Package And File Plan

### Task 1. Add dependency parsing package

Purpose:

- parse dependency trailers from raw commit messages
- normalize ticket IDs
- dedupe dependency entries

Files:

- new `internal/dependency/parser.go`
- new `internal/dependency/model.go`
- new `internal/dependency/parser_test.go`

Implementation notes:

- keep trailer parsing strict and line-based
- accept only supported trailer keys in the first slice
- reuse `ticket.Parser` for validation and normalization

Tests:

- single `Depends-On` trailer
- repeated trailer lines
- comma-separated dependency values
- mixed-case trailer keys
- duplicate dependency values collapse into one entry
- invalid ticket-looking values are ignored

### Task 2. Add optional commit-message loading to adapters

Purpose:

- allow services to fetch raw commit bodies without widening the base adapter contract

Files:

- update `internal/scm/git/adapter.go`
- update `internal/scm/git/adapter_test.go`
- optional helper in new `internal/scm/git/commit_messages.go`

Implementation notes:

- add an internal extension interface such as `commitMessageProvider`
- implement it in Git only
- do not change `internal/scm/types.go` unless the final implementation truly needs a shared type
- keep SVN behavior untouched for this slice

Tests:

- commit messages are loaded for unique hashes only once
- returned bodies preserve trailer lines
- unsupported adapters degrade cleanly

### Task 3. Enrich inspect results with dependency evidence

Purpose:

- make declared dependencies visible before branch comparison starts

Files:

- update `internal/inspect/service.go`
- new `internal/inspect/dependency.go`
- update `internal/inspect/service_test.go`

Implementation notes:

- extend `RepositoryInspection` with declared dependency data
- keep file-risk inference and dependency inference separate, then combine them in the result
- do not make `inspect` depend on `from/to` branch inputs

Tests:

- inspect returns declared dependencies for a ticket
- repositories without dependency trailers still render normally
- Git and non-Git adapters continue to behave predictably

### Task 4. Resolve dependency status in plan and verify

Purpose:

- turn declared dependency evidence into release reasoning

Files:

- new `internal/dependency/resolver.go`
- new `internal/dependency/resolver_test.go`
- update `internal/plan/service.go`
- update `internal/plan/service_test.go`

Implementation notes:

- resolve dependencies once per ticket plan, not separately for each repository loop
- evaluate against the full discovered repository set
- convert `missing-target` into:
  - blocked risk signal
  - release action to include the companion ticket
  - verification reason
- convert `unresolved` into:
  - warning risk signal
  - note that workspace or metadata coverage may be incomplete

Tests:

- dependency already in target produces no extra risk
- dependency in source but missing in target blocks the plan
- unresolved dependency produces warning but not blocked verdict
- multiple commits declaring the same dependency do not duplicate output
- one ticket depending on several companion tickets aggregates correctly

### Task 5. Surface dependency results in release packet and terminal output

Purpose:

- make dependency reasoning visible to humans and scripts

Files:

- update `internal/manifest/service.go`
- update `internal/output/inspect.go`
- update `internal/output/plan.go`
- update `internal/output/manifest.go`
- update `internal/output/human.go` if shared formatting helpers are needed

Implementation notes:

- add a dependency section to inspect output
- add dependency status and risk to plan output
- include dependency summary in manifest highlights and repository details
- structured output should expose dependency status directly instead of forcing consumers to parse risk text

Tests:

- update `internal/cli/app_test.go`
- update `internal/cli/testdata/inspect.golden`
- update `internal/cli/testdata/plan.golden`
- update `internal/cli/testdata/plan_json.golden`
- update `internal/cli/testdata/verify.golden`
- update `internal/cli/testdata/manifest_generate.golden`

If JSON output changes materially, add new dedicated contract fixtures instead of only mutating existing golden files.

### Task 6. Update docs and examples with the new behavior

Purpose:

- document the supported metadata and make the new risk legible to teams

Files:

- update `docs/05-domain-model.md`
- update `docs/08-commit-convention.md`
- update `docs/03-cli-spec.md`
- update `docs/11-test-strategy.md`
- update `examples/README.md` if a dependency example is added

Tests:

- doc examples match real CLI output
- dependency trailer examples use supported syntax only

## Suggested Delivery Order

Implement in this order:

1. `internal/dependency` parsing
2. Git commit-message loading
3. inspect-level dependency evidence
4. plan and verify dependency resolution
5. manifest and output updates
6. docs and example refresh

This order keeps the risk logic grounded in real parser behavior before output work starts.

## JSON Contract Guidance

This slice should improve structured output instead of hiding dependency logic inside prose.

Recommended additions:

- inspect result includes declared dependencies
- plan result includes dependency resolutions and dependency-derived risk signals
- verify result includes dependency-derived reasons
- manifest packet includes dependency summaries and per-repo dependency details where relevant

Do not rely on parsing the human-readable summary strings in downstream tooling.

## Acceptance Criteria

This slice is done when:

- `Depends-On` trailers are parsed reliably from Git commit bodies
- dependency evidence appears in `gig inspect`
- a confirmed dependency missing from the target branch blocks `gig plan` and `gig verify`
- unresolved dependency metadata produces a warning instead of silent omission
- manifest output mentions dependency findings clearly
- golden and unit tests cover parser, resolver, and CLI output behavior
- the code remains read-only and introduces no hidden write action

## Follow-Up Work After This Slice

The next slices after this one should be:

1. Jira, PR, and deployment evidence
2. richer multi-ticket release bundles
3. controlled promote execution after strong dry-run and approval flows

That sequencing keeps `gig` on the path from branch-aware reasoning toward evidence-aware release planning.
