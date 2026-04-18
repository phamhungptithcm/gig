# Product Reset Audit

## Why This Audit Exists

`gig` already has useful release logic, but the current product shape is still too hard to adopt.

The core issue is not quality of the read-only analysis.
The issue is the entry path:

- too local-workspace-first
- too config-first
- too branch-flag-heavy
- not source-control-native enough for real daily usage

This document proposes a reset in usage design and internal structure while keeping the strongest parts of the current direction.

Status update:

- GitHub login-backed remote audit is live
- GitLab login-backed remote audit is live
- Bitbucket login-backed remote audit is live
- Azure DevOps login-backed remote audit is live
- remote SVN is live
- initial workarea save and switch flow is live
- interactive workarea and repository pickers with recent-history ranking are live
- running `gig` with no subcommand now opens a guided front door
- successful remote runs can now save inferred branch topology back into the current workarea
- `gig assist doctor` now checks whether the bundled DeerFlow sidecar is configured, startable, and reachable
- `gig assist setup` now bootstraps the bundled DeerFlow sidecar config

## Executive Summary

Keep:

- ticket-aware release reasoning
- risk signals and manual-step generation
- release snapshots, plans, and manifests as internal capabilities
- multi-repo and mixed Git or SVN support as a long-term requirement

Change:

- move from Homebrew and Scoop distribution to a public npm-first install path with direct-installer fallback
- move from local-first to remote-first
- move from config-required to zero-config default
- move from manual branch mapping to protected-branch discovery plus override
- move from many workflow-shaped commands to a smaller set of user-intent commands
- move from raw SCM adapters only to provider-aware connections with login and capability checks

Recommended new positioning:

`gig is a source-control-native ticket reconciliation and release decision tool.`

## What The Current Project Gets Right

The existing code and docs already contain the right release question:

`Did we miss any change for this ticket?`

The current implementation is strong in these areas:

- ticket-aware commit discovery
- cross-repo inspection
- branch-to-branch comparison
- risk classification
- repeatable snapshots for audit
- human and JSON output

Those should be preserved.

## Main Findings

### 1. The product is still workspace-first instead of source-control-first

Current activation assumes the user has a local workspace and points commands at `--path .`.

That is visible in:

- the README first workflow
- the CLI guide examples
- repository detection through local filesystem walking
- SCM adapters that shell out to local `git` and `svn`

This is the biggest mismatch with the intended direction.
Many real users want to point `gig` at GitHub, GitLab, Bitbucket, or SVN directly and inspect release state without cloning or wiring a special workspace first.

### 2. Config carries too much weight in the first-use path

The current config is useful for mature teams, but it is too central for activation.

Today config is expected to solve:

- environment-to-branch mapping
- repository catalog
- service naming
- owner metadata
- repo kind metadata
- repo notes

That helps output quality later, but it makes first-run adoption heavier than it should be.

The product currently treats config as a prerequisite for trust.
The new direction should treat config as an optional quality upgrade.

### 3. Branch and environment modeling is too manual

The current flow still assumes the user knows:

- the source branch
- the target branch
- the environment mapping

This causes repeated `--from`, `--to`, and `--envs` usage.

That is acceptable for power users.
It is not acceptable as the main workflow.

The tool should instead:

- detect protected branches from the source-control provider
- exclude short-lived branches and sub-branches by heuristic
- infer likely environment order
- let the user override only when detection is wrong

### 4. The command surface is broader than the user intent surface

The current command set is internally coherent, but externally it exposes too much of the engine shape:

- `scan`
- `find`
- `inspect`
- `env status`
- `plan`
- `verify`
- `manifest`
- `snapshot create`
- `doctor`
- `resolve`

For a new user, this is more decision-making than necessary.

The product should expose fewer top-level decisions:

- connect
- inspect ticket
- check promotion
- capture release
- generate release bundle

The current commands can still exist, but they should become implementation-level or advanced-mode commands.

### 5. Authentication and provider access need deeper product UX

The old problem was that SCM adapters existed without a true provider/session layer.
That is no longer fully true.
`gig` now has login-backed remote access across GitHub, GitLab, Bitbucket, Azure DevOps, and remote SVN.

The remaining gap is product polish around:

- workarea-driven project picking
- repository or organization discovery
- richer provider capability display
- issue-tracker evidence retrieval

Without those additions, the product is not yet fully source-control-native at the UX layer.

### 6. Powerful release abstractions arrived before the activation path was simplified

Snapshots, release plans, and release bundles are useful.
But they increase conceptual weight before the product has solved:

- easy connection
- zero-config first run
- branch auto-detection
- provider-native evidence

These features should stay, but they should sit behind a simpler front door.

## New Product Direction

The correct direction is:

`remote-first, local-optional, provider-aware, zero-config by default`

That means:

- the default mode talks to GitHub, GitLab, Bitbucket, or SVN directly
- local repository scanning becomes an optional fallback mode
- provider login is part of the product, not an external prerequisite the user has to infer
- protected branches become the default source of environment understanding
- config becomes optional metadata, not a gate to basic usefulness

If AI assist is added, it should sit on top of deterministic release evidence.
The model can summarize, explain, and point to next actions, but it should not replace the core ticket, branch, and risk reasoning.
The current assist slice now supports audience-specific ticket briefs, release-level briefs from saved snapshots or live ticket sets, and conflict-resolution briefs for active Git conflicts, which is the right shape as long as the bundle remains the ground truth.

## New User Experience

### 1. Connect once

Examples:

```bash
gig login github
gig login gitlab
gig login bitbucket
gig login svn
```

Behavior:

- detect whether credentials already exist
- if not, launch the provider login flow
- store a local session profile
- verify basic API and repository access before moving on

### 2. Point at a repo or organization, not a local workspace

Examples:

```bash
gig connect github:org/repo
gig connect gitlab:group/project
gig connect bitbucket:workspace/repo
gig connect svn:https://svn.example.com/repos/app
```

Optional local mode:

```bash
gig connect --path .
```

### 3. Inspect a ticket with zero required branch flags

Examples:

```bash
gig inspect ABC-123
gig inspect ABC-123 --repo github:org/repo
gig inspect ABC-123 --scope release-2026-04-09
```

Default behavior:

- search the connected repos
- detect the main protected branches
- show current branch topology
- show dependencies, risks, and candidate follow-up changes

### 4. Check whether promotion is safe using target intent, not raw branch pairs

Examples:

```bash
gig check ABC-123 --target staging
gig check ABC-123 --target production
gig check ABC-123 --from staging --to main
```

Default behavior:

- resolve target branch from detected branch topology
- infer the source branch when possible
- ask for override only when the topology is ambiguous

### 5. Capture and package release state from connected evidence

Examples:

```bash
gig release capture rel-2026-04-09 --tickets-file tickets.txt
gig release plan rel-2026-04-09
gig release verify rel-2026-04-09
gig release bundle rel-2026-04-09
```

The snapshot or bundle layer stays.
The difference is that it should build from connected provider evidence by default instead of assuming a pre-arranged local workspace.

## New Internal Architecture

## 1. Provider Layer

Add a new first-class layer above SCM adapters.

Responsibilities:

- authentication
- session persistence
- repository lookup
- protected branch lookup
- PR or merge request lookup
- deployment or pipeline lookup

Example providers:

- GitHub
- GitLab
- Bitbucket
- SVN server session

## 2. Repository Source Layer

Unify the source of repository evidence behind one model:

- remote provider repository
- local Git repo
- local SVN working copy
- remote SVN URL

This becomes the core input to inspection and planning.

## 3. Branch Topology Resolver

Add a dedicated branch resolver that understands:

- protected branches
- common naming patterns such as `dev`, `develop`, `test`, `staging`, `uat`, `main`, `master`, `release/*`
- ignored short-lived branches such as feature, fix, hotfix, or personal sub-branches
- optional manual overrides

This should replace the current assumption that users will always supply `--from`, `--to`, or `--envs`.

## 4. Evidence Graph

Build one normalized graph per ticket or release:

- commits
- branches
- PRs or merge requests
- deployments
- related tickets
- manual-step evidence
- repository metadata

Current `inspect`, `plan`, `verify`, and `manifest` logic should consume this graph rather than querying each layer independently.

## 5. Decision Engine

Keep the current strengths here:

- missing-commit detection
- dependency risk detection
- manual-review generation
- release packet generation

But make this engine independent from whether evidence came from:

- local Git
- local SVN
- GitHub API
- GitLab API
- Bitbucket API
- remote SVN queries

## Config Reset

### Default Rule

No config file should be required for first success.

### New Split

Use two optional configuration types:

1. user profile config

- stored in the user's config directory
- contains provider sessions, defaults, and branch override preferences

2. workspace metadata file

- optional project-level file such as `.gig/workspace.yaml`
- contains service labels, owners, repo kinds, and manual notes

This keeps high-value metadata without making it the activation bottleneck.

### What Should Move Out Of The First-Run Path

- mandatory environment mapping
- mandatory repo catalog coverage
- repeated `--path`
- repeated `--config`
- repeated `--envs`

## Authentication Model

Authentication should be explicit and guided.

When the user runs a provider-backed command without a valid session, `gig` should:

1. detect the provider from the repo target or URL
2. check whether a valid session already exists
3. if not, launch the appropriate login flow
4. verify that the user can access the requested repository or organization
5. continue the original command automatically

Examples:

- GitHub: prefer `gh auth status` and `gh auth login`, or direct OAuth later
- GitLab: prefer `glab auth status` and `glab auth login`, or direct token flow later
- Bitbucket: support API token login, with macOS Keychain storage where available
- SVN: validate `svn info` or `svn ls`, then prompt for credentials only when required

This is much closer to the product behavior users expect.

## Branch Detection Strategy

Protected branches should become the default input for environment discovery.

Recommended order:

1. load protected branches from provider metadata when available
2. rank branch names by known environment heuristics
3. remove branches that look like feature or personal branches
4. infer promotion order
5. ask for a manual mapping only if the result is ambiguous
6. cache the resolved mapping as a user or workspace override

This approach matches the user's mental model better than a required `gig.yaml` file up front.

## Command Reset

Recommended primary commands:

- `gig login <provider>`
- `gig connect <target>`
- `gig inspect <ticket-id>`
- `gig check <ticket-id> --target <env-or-branch>`
- `gig release capture <release-id>`
- `gig release plan <release-id>`
- `gig release verify <release-id>`
- `gig release bundle <release-id>`

Recommended advanced commands:

- `gig doctor`
- `gig resolve`
- explicit branch-pair mode
- explicit local workspace mode

This keeps power-user control without forcing every user to learn the full internal model on day one.

## Migration Plan

### Phase 0: Stop Growing The Old Activation Model

Do not add more first-run dependence on:

- config
- local workspace assumptions
- manual env flags

### Phase 1: Introduce Provider Sessions

Implement:

- provider abstraction
- login flow
- session storage
- repository targeting by URL or provider handle

### Phase 2: Add Remote-First Inspect

Implement:

- remote commit search
- remote protected branch discovery
- remote branch comparison where supported
- local fallback when remote APIs are not enough

### Phase 3: Add Branch Topology Detection

Implement:

- protected-branch inference
- environment ranking
- override persistence

### Phase 4: Rebase Existing Planning Logic Onto The New Evidence Graph

Reuse current logic for:

- risks
- dependencies
- manual steps
- snapshots
- manifests

But feed it from the new provider-aware evidence layer.

### Phase 5: Keep Local Mode As Advanced Or Air-Gapped Mode

Local mode should remain valuable for:

- monorepo workspaces
- air-gapped enterprise environments
- teams that prefer local Git or SVN truth

But it should no longer define the main product shape.

## Final Recommendation

Do not throw away the release logic.

Do reset the product around:

- source-control-native access
- zero-config first success
- protected-branch auto-detection
- guided login and repository connection
- local mode as fallback, not as the default worldview

That is the cleanest path to make `gig` feel practical instead of academically correct.
