# SCM Abstraction

## Purpose

The SCM abstraction keeps the rest of the system independent from Git or SVN details.

This means the CLI and services can ask simple questions like:

- is this a repo?
- what branch is this repo on?
- which commits match this ticket?
- what is missing between two branches?

## Standard Adapter Interface

An adapter should support these capabilities:

- detect whether a path is a repository root
- detect the enclosing repository root from any child path
- return current branch or working line information
- search commits by ticket ID
- compare two branches for one ticket
- prepare future cherry-pick or promotion support

## Example Contract

```go
type Adapter interface {
    Type() Type
    DetectRoot(path string) (string, bool, error)
    IsRepository(path string) (bool, error)
    CurrentBranch(ctx context.Context, repoRoot string) (string, error)
    SearchCommits(ctx context.Context, repoRoot string, query SearchQuery) ([]Commit, error)
    CompareBranches(ctx context.Context, repoRoot string, query CompareQuery) (CompareResult, error)
    PrepareCherryPick(ctx context.Context, repoRoot string, plan CherryPickPlan) error
}
```

## Why This Matters

- the CLI does not need Git-only code
- the service layer can stay simple
- Git can be shipped first
- SVN can be added later without a redesign
- tests can use fake adapters instead of real repos

## Git Notes

The Git adapter:

- detects repositories through `.git`
- searches commits with `git log`
- compares branches with `git cherry` plus ticket filtering
- resolves branch lists for display with `git for-each-ref`

## SVN Notes

The SVN adapter is a prepared stub today.

It can:

- detect `.svn`

It does not yet:

- search history
- compare branches
- prepare promotion

That work belongs to a later phase.
