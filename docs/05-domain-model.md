# Domain Model

## Repo

Represents one detected repository in the workspace.

Fields:

- `Name`
- `Root`
- `Type`
- `CurrentBranch`

## CommitRef

Represents one commit that the tool found and wants to show or compare.

Fields:

- `Hash`
- `Subject`
- `Branches`

Notes:

- in Git, `Hash` is the commit SHA
- `Branches` is optional because branch data may not always be cheap or available

## TicketChangeSet

Represents all commits for one ticket in one repository.

Suggested fields:

- `TicketID`
- `Repo`
- `Commits []CommitRef`

Use:

- output of `find`
- input to promotion planning

## BranchDiff

Represents the difference for one ticket between two branches in one repository.

Suggested fields:

- `TicketID`
- `Repo`
- `FromBranch`
- `ToBranch`
- `SourceCommits []CommitRef`
- `TargetCommits []CommitRef`
- `MissingCommits []CommitRef`

Use:

- output of `diff`
- input to promotion planning

## PromotionPlan

Represents what the tool plans to do during a future promote flow.

Suggested fields:

- `TicketID`
- `FromBranch`
- `ToBranch`
- `RepoPlans`
- `DryRun`
- `Warnings`

Each repo plan may include:

- repository info
- commits to apply
- dependency warnings
- risk warnings

## PromotionResult

Represents the result after a future promotion run.

Suggested fields:

- `TicketID`
- `FromBranch`
- `ToBranch`
- `Applied`
- `Skipped`
- `Failed`
- `Warnings`
- `Errors`

## Query Models Used Today

The current implementation also uses simple transport models:

### SearchQuery

Used to search commits for a ticket.

Fields:

- `TicketID`
- `Branch`

### CompareQuery

Used to compare ticket-related commits between branches.

Fields:

- `TicketID`
- `FromBranch`
- `ToBranch`

### CompareResult

Current normalized adapter result.

Fields:

- `SourceCommits`
- `TargetCommits`
- `MissingCommits`

## Why These Models Matter

- they make the service layer easier to reason about
- they keep output and SCM code from inventing their own shapes
- they prepare the codebase for promote, snapshot, and dependency features
